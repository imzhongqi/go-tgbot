package tgbot

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/panjf2000/ants/v2"
)

// Command is telegram command
type Command struct {
	Name        string
	Description string
	Hide        bool // hide the command on telegram commands menu
	Handler     Handler
}

func (c Command) String() string {
	return fmt.Sprintf("/%s - %s", c.Name, c.Description)
}

// UpdatesHandler handler another update
type UpdatesHandler func(ctx *Context)

// Handler command handler
type Handler func(ctx *Context) error

// ErrHandler error handler
type ErrHandler func(err error)

// Bot wrapper the telegram bot
type Bot struct {
	wg     sync.WaitGroup
	ctx    context.Context
	cancel func()

	api *tgbotapi.BotAPI

	commands    []*Command
	cmdHandlers map[string]Handler

	undefinedCommandHandler Handler
	errHandler              ErrHandler
	updatesHandler          UpdatesHandler
	panicHandler            func(interface{}) (message string)

	workerNum  int
	workerPool *ants.Pool

	// updateC chan buffer size
	bufSize int
	updateC chan tgbotapi.Update

	timeout        int
	limit          int
	offset         int
	allowedUpdates []string
}

func NewBot(api *tgbotapi.BotAPI, opts ...Option) *Bot {
	bot := &Bot{
		api:         api,
		cmdHandlers: make(map[string]Handler),
		timeout:     60,
		errHandler:  func(err error) {},
		workerNum:   runtime.GOMAXPROCS(0),
		limit:       100,
		ctx:         context.Background(),
	}

	bot.panicHandler = func(v interface{}) string {
		if v != nil {
			bot.errHandler(fmt.Errorf("tgbot panic: %v", v))
		}
		return "oops! Service is temporarily unavailable"
	}

	for _, o := range opts {
		o(bot)
	}

	bot.ctx, bot.cancel = context.WithCancel(bot.ctx)

	// hijack the api client
	bot.api.Client = &client{cli: bot.api.Client, ctx: bot.ctx}

	// set the updateC size for pollUpdates
	if bot.bufSize == 0 {
		bot.bufSize = bot.limit
	}
	bot.updateC = make(chan tgbotapi.Update, bot.bufSize)

	return bot
}

func (bot *Bot) AddCommand(cmd *Command) {
	bot.commands = append(bot.commands, cmd)
	bot.cmdHandlers[cmd.Name] = cmd.Handler
}

func (bot *Bot) Commands() []*Command {
	commands := make([]*Command, 0, len(bot.commands))
	for _, cmd := range bot.commands {
		if !cmd.Hide {
			commands = append(commands, cmd)
		}
	}
	return commands
}

func (bot *Bot) setupCommands() error {
	commands := make([]tgbotapi.BotCommand, 0, len(bot.commands))
	for _, hdr := range bot.Commands() {
		commands = append(commands, tgbotapi.BotCommand{
			Command:     hdr.Name,
			Description: hdr.Description,
		})
	}

	_, err := bot.api.Request(tgbotapi.NewSetMyCommands(commands...))
	return err
}

func (bot *Bot) handleUpdate(update tgbotapi.Update) {
	if bot.workerPool == nil || bot.panicHandler != nil {
		defer func() {
			if e := recover(); e != nil {
				if tipMessage := bot.panicHandler(e); tipMessage != "" {
					msg := tgbotapi.NewMessage(update.FromChat().ID, tipMessage)
					msg.DisableWebPagePreview = true
					if _, err := bot.api.Send(msg); err != nil {
						bot.errHandler(err)
					}
				}
			}
		}()
	}

	switch {
	case update.Message != nil && update.Message.IsCommand():
		handler, ok := bot.cmdHandlers[update.Message.Command()]
		if !ok {
			handler = bot.undefinedCmdHandler
		}

		// use workerPool if workerPool available
		if bot.workerPool != nil {
			if err := bot.workerPool.Submit(func() {
				if err := handler(&Context{
					Context: bot.ctx,
					BotAPI:  bot.api,
					Message: update.Message,
					Update:  &update,
				}); err != nil {
					bot.errHandler(err)
				}
			}); err != nil {
				bot.errHandler(err)
			}
			return
		}

		if err := handler(&Context{
			Context: bot.ctx,
			BotAPI:  bot.api,
			Message: update.Message,
			Update:  &update,
		}); err != nil {
			bot.errHandler(err)
		}

	default:
		if bot.updatesHandler != nil {
			if bot.workerPool != nil {
				if err := bot.workerPool.Submit(func() {
					bot.updatesHandler(&Context{
						Context: bot.ctx,
						BotAPI:  bot.api,
						Message: update.Message,
						Update:  &update,
					})
				}); err != nil {
					bot.errHandler(err)
				}
				return
			}

			bot.updatesHandler(&Context{
				Context: bot.ctx,
				BotAPI:  bot.api,
				Message: update.Message,
				Update:  &update,
			})
		}
	}
}

func (bot *Bot) undefinedCmdHandler(ctx *Context) error {
	if bot.undefinedCommandHandler != nil {
		return bot.undefinedCommandHandler(ctx)
	}
	return ctx.ReplyText("Unrecognized command!!!")
}

func (bot *Bot) startWorker() {
	startWorker := func() {
		defer bot.wg.Done()

		for {
			select {
			case <-bot.ctx.Done():
				return

			case update := <-bot.updateC:
				bot.handleUpdate(update)
			}
		}
	}

	for i := 0; i < bot.workerNum; i++ {
		bot.wg.Add(1)
		go startWorker()
	}
}

func (bot *Bot) pollUpdates() {
	for {
		select {
		case <-bot.ctx.Done():
			return

		default:
		}

		updates, err := bot.api.GetUpdates(tgbotapi.UpdateConfig{
			Limit:          bot.limit,
			Offset:         bot.offset,
			Timeout:        bot.timeout,
			AllowedUpdates: bot.allowedUpdates,
		})
		if err != nil && !errors.Is(err, context.Canceled) {
			bot.errHandler(fmt.Errorf("failed to get updates, error: %w", err))
			time.Sleep(3 * time.Second)
			continue
		}

		for _, update := range updates {
			if update.UpdateID >= bot.offset {
				bot.offset = update.UpdateID + 1
				bot.updateC <- update
			}
		}
	}
}

func (bot *Bot) Run() error {
	// setup bot commands
	if err := bot.setupCommands(); err != nil {
		return fmt.Errorf("failed to setup commands, error: %w", err)
	}

	// start the worker
	bot.startWorker()

	// start poll updates
	go bot.pollUpdates()

	// wait all worker done
	bot.wg.Wait()

	return nil
}

func (bot *Bot) Stop() {
	bot.cancel()

	bot.wg.Wait()
}
