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

// UpdatesHandler handler another update
type UpdatesHandler func(ctx *Context)

// Handler command handler
type Handler func(ctx *Context) error

// ErrHandler error handler
type ErrHandler func(err error)

// Bot wrapper the telegram bot
type Bot struct {
	api *tgbotapi.BotAPI

	wg sync.WaitGroup

	pool sync.Pool

	ctx    context.Context
	cancel context.CancelFunc

	commands    []*Command
	cmdHandlers map[string]Handler

	timeout                 time.Duration
	undefinedCommandHandler Handler
	errHandler              ErrHandler
	updatesHandler          UpdatesHandler
	panicHandler            func(interface{}) (message string)

	workerNum  int
	workerPool *ants.Pool

	// updateC chan buffer size
	bufSize int
	updateC chan *tgbotapi.Update

	updateTimeout  int
	limit          int
	offset         int
	allowedUpdates []string
}

func NewBot(api *tgbotapi.BotAPI, opts ...Option) *Bot {
	bot := &Bot{
		ctx: context.Background(),

		api: api,

		cmdHandlers: make(map[string]Handler),
		errHandler:  func(err error) {},

		workerNum: runtime.GOMAXPROCS(0),

		updateTimeout: 60,
		limit:         100,
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
	bot.updateC = make(chan *tgbotapi.Update, bot.bufSize)

	return bot
}

func (bot *Bot) allocateContext() *Context {
	if v := bot.pool.Get(); v != nil {
		ctx := v.(*Context)
		ctx.Context = bot.ctx
		return ctx
	}
	return &Context{
		Context: bot.ctx,
		BotAPI:  bot.api,
		bot:     bot,
	}
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

func (bot *Bot) handleUpdate(update *tgbotapi.Update) {
	ctx := bot.allocateContext()
	ctx.update = update

	if bot.workerPool == nil || bot.panicHandler != nil {
		defer func() {
			if e := recover(); e != nil {
				if tipMessage := bot.panicHandler(e); tipMessage != "" {
					if err := ctx.ReplyText(tipMessage); err != nil {
						bot.errHandler(err)
					}
				}
			}
		}()
	}

	executeHandler := func() {
		if bot.timeout > 0 {
			var cancel context.CancelFunc
			ctx.Context, cancel = context.WithTimeout(ctx.Context, bot.timeout)
			defer cancel()
		}

		switch {
		case ctx.IsCommand():
			bot.executeCommandHandler(ctx)

		default:
			bot.executeUpdatesHandler(ctx)
		}

		ctx.put()
	}

	if bot.workerPool != nil && !bot.workerPool.IsClosed() {
		if err := bot.workerPool.Submit(executeHandler); err != nil {
			bot.errHandler(err)
		}
		return
	}

	executeHandler()
}

func (bot *Bot) executeCommandHandler(ctx *Context) {
	handler, ok := bot.cmdHandlers[ctx.Command()]
	if !ok {
		handler = bot.undefinedCmdHandler
	}

	if err := handler(ctx); err != nil {
		bot.errHandler(err)
	}
}

func (bot *Bot) executeUpdatesHandler(ctx *Context) {
	if bot.updatesHandler == nil {
		return
	}

	bot.updatesHandler(ctx)
}

func (bot *Bot) undefinedCmdHandler(ctx *Context) error {
	if bot.undefinedCommandHandler != nil {
		return bot.undefinedCommandHandler(ctx)
	}
	return ctx.ReplyText("Unrecognized command!!!")
}

func (bot *Bot) startWorkers() {
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
			Timeout:        bot.updateTimeout,
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
				bot.updateC <- &update
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
	bot.startWorkers()

	// start poll updates
	go bot.pollUpdates()

	// wait all worker done
	bot.wg.Wait()

	return nil
}

func (bot *Bot) Stop() context.Context {
	bot.cancel()
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		bot.wg.Wait()
		cancel()
	}()
	return ctx
}
