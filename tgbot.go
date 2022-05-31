package tgbot

import (
	"context"
	"errors"
	"fmt"
	"sync"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Bot wrapper the telegram bot.
type Bot struct {
	api *tgbotapi.BotAPI

	// opts is bot options
	opts *Options

	wg sync.WaitGroup

	pool sync.Pool

	ctx    context.Context
	cancel context.CancelFunc

	commands    map[CommandScope][]*Command
	cmdHandlers map[string]Handler

	updateC chan *tgbotapi.Update
}

func NewBot(api *tgbotapi.BotAPI, opts ...Option) *Bot {
	bot := &Bot{
		opts: newOptions(opts...),
		api:  api,
	}

	bot.ctx, bot.cancel = context.WithCancel(bot.opts.ctx)

	// hijack the api client.
	bot.api.Client = &client{cli: bot.api.Client, ctx: bot.ctx}

	// set the updateC size for pollUpdates.
	if bot.opts.bufSize == 0 {
		bot.opts.bufSize = bot.opts.limit
	}
	bot.updateC = make(chan *tgbotapi.Update, bot.opts.bufSize)

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

func (bot *Bot) AddCommands(commands ...*Command) {
	if bot.cmdHandlers == nil || bot.commands == nil {
		bot.cmdHandlers = make(map[string]Handler)
		bot.commands = make(map[CommandScope][]*Command)
	}

	for _, c := range commands {
		switch {
		case c.Name == "":
			panic("command name must be non-empty")
		case c.Description == "":
			panic("command description must be non-empty")
		case c.Handler == nil:
			panic("command handler must be non-nil")
		}
		if _, ok := bot.cmdHandlers[c.Name]; ok {
			panic("duplicate command name: " + c.Name)
		}

		bot.cmdHandlers[c.Name] = c.Handler
		if c.Scopes == nil {
			c.Scopes = append(c.Scopes, CommandScope{invalid: true})
		}
		for _, scope := range c.Scopes {
			bot.commands[scope] = append(bot.commands[scope], c)
		}
	}
}

func (bot *Bot) Commands() map[CommandScope][]*Command {
	return bot.commands
}

func (bot *Bot) setupCommands() error {
	if !bot.opts.autoSetupCommands {
		return nil
	}

	for scope, cmds := range bot.Commands() {
		commands := make([]tgbotapi.BotCommand, 0, len(bot.commands))
		for _, cmd := range cmds {
			commands = append(commands, tgbotapi.BotCommand{
				Command:     cmd.Name,
				Description: cmd.Description,
			})
		}

		if len(commands) == 0 {
			continue
		}

		cmd := tgbotapi.NewSetMyCommands(commands...)
		if !scope.invalid {
			cmd = tgbotapi.NewSetMyCommandsWithScope(scope.toScope(), commands...)
			cmd.LanguageCode = scope.LanguageCode
		}
		if _, err := bot.api.Request(cmd); err != nil {
			return err
		}
	}

	return nil
}

func (bot *Bot) handleUpdate(update *tgbotapi.Update) {
	ctx := bot.allocateContext()
	ctx.update = update

	updateHandler := func() {
		if bot.opts.panicHandler != nil {
			defer func() {
				if e := recover(); e != nil {
					bot.opts.panicHandler(ctx, e)
				}
			}()
		}

		if bot.opts.timeout > 0 {
			var cancel context.CancelFunc
			ctx.Context, cancel = context.WithTimeout(ctx.Context, bot.opts.timeout)
			defer cancel()
		}

		switch {
		case bot.cmdHandlers != nil && ctx.IsCommand():
			bot.commandHandler(ctx)

		default:
			bot.updatesHandler(ctx)
		}

		ctx.put()
	}

	if bot.opts.workerPool != nil && !bot.opts.workerPool.IsClosed() {
		if err := bot.opts.workerPool.Submit(updateHandler); err != nil {
			bot.opts.errHandler(err)
		}
		return
	}

	updateHandler()
}

func (bot *Bot) commandHandler(ctx *Context) {
	handler, ok := bot.cmdHandlers[ctx.Command()]
	if !ok {
		handler = bot.undefinedCmdHandler
	}

	if err := handler(ctx); err != nil {
		bot.opts.errHandler(err)
	}
}

func (bot *Bot) updatesHandler(ctx *Context) {
	if bot.opts.updatesHandler == nil {
		return
	}

	bot.opts.updatesHandler(ctx)
}

func (bot *Bot) undefinedCmdHandler(ctx *Context) error {
	if bot.opts.undefinedCommandHandler != nil {
		return bot.opts.undefinedCommandHandler(ctx)
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

	for i := 0; i < bot.opts.workerNum; i++ {
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
			Limit:          bot.opts.limit,
			Offset:         bot.opts.offset,
			Timeout:        bot.opts.updateTimeout,
			AllowedUpdates: bot.opts.allowedUpdates,
		})
		if err != nil && !errors.Is(err, context.Canceled) {
			bot.opts.pollUpdatesErrorHandler(err)
			continue
		}

		for _, update := range updates {
			if update.UpdateID >= bot.opts.offset {
				bot.opts.offset = update.UpdateID + 1
				bot.updateC <- &update
			}
		}
	}
}

func (bot *Bot) Run() error {
	// setup bot commands.
	if err := bot.setupCommands(); err != nil {
		return fmt.Errorf("failed to setup commands, error: %w", err)
	}

	// start the worker.
	bot.startWorkers()

	// start poll updates.
	go bot.pollUpdates()

	// wait all worker done.
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
