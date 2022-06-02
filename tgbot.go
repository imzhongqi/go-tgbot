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
	opts *options

	wg sync.WaitGroup

	pool sync.Pool

	ctx    context.Context
	cancel context.CancelFunc

	commands    map[CommandScope][]*Command
	cmdHandlers map[string]Handler

	updateC chan *tgbotapi.Update

	err error
}

// NewBot new a telegram bot.
func NewBot(api *tgbotapi.BotAPI, opts ...Option) *Bot {
	o := newOptions(opts...)

	ctx, cancel := context.WithCancel(o.ctx)

	// set the updateC size for pollUpdates.
	if o.bufSize == 0 {
		o.bufSize = o.limit
	}

	return &Bot{
		api:     api,
		opts:    o,
		ctx:     ctx,
		cancel:  cancel,
		updateC: make(chan *tgbotapi.Update, o.bufSize),
	}
}

func (bot *Bot) allocateContextWithUpdate(update *tgbotapi.Update) (c *Context, recycle func()) {
	var (
		ctx    = bot.ctx
		cancel context.CancelFunc
	)
	if bot.opts.timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, bot.opts.timeout)
	}

	recycle = func() {
		if cancel != nil {
			cancel()
		}

		c.reset()
		bot.pool.Put(c)
	}

	if v := bot.pool.Get(); v != nil {
		c = v.(*Context)
		c.Context = ctx
		c.update = update
		return c, recycle
	}

	return &Context{
		Context: ctx,
		BotAPI:  bot.api,
		update:  update,
	}, recycle
}

// AddCommands add commands to the bot.
func (bot *Bot) AddCommands(commands ...*Command) {
	if bot.cmdHandlers == nil {
		bot.cmdHandlers = make(map[string]Handler)
	}
	if bot.commands == nil {
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
		if len(c.Scopes) == 0 {
			c.Scopes = append(c.Scopes, noScope)
		}

		// used to filter duplicate scope.
		scopes := make(map[CommandScope]struct{})

		for _, scope := range c.Scopes {
			if _, ok := scopes[scope]; ok {
				continue
			}
			scopes[scope] = struct{}{}

			bot.commands[scope] = append(bot.commands[scope], c)
		}
	}
}

func (bot *Bot) Commands() map[CommandScope][]*Command {
	return bot.commands
}

func (bot *Bot) setupCommands() error {
	if bot.opts.disableAutoSetupCommands {
		return nil
	}

	for scope, commands := range bot.Commands() {
		botCommands := make([]tgbotapi.BotCommand, 0, len(commands))
		for _, cmd := range commands {
			if cmd.Hide {
				continue
			}

			botCommands = append(botCommands, tgbotapi.BotCommand{
				Command:     cmd.Name,
				Description: cmd.Description,
			})
		}

		if len(botCommands) == 0 {
			continue
		}

		cmd := tgbotapi.NewSetMyCommands(botCommands...)
		if scope != nil && scope != noScope {
			cmd = tgbotapi.NewSetMyCommandsWithScopeAndLanguage(tgbotapi.BotCommandScope{
				Type:   scope.Type(),
				ChatID: scope.ChatID(),
				UserID: scope.UserID(),
			}, scope.LanguageCode(), botCommands...)
		}
		if _, err := bot.api.Request(cmd); err != nil {
			return err
		}
	}

	return nil
}

func (bot *Bot) makeUpdateHandler(update *tgbotapi.Update) func() {
	return func() {
		ctx, recycle := bot.allocateContextWithUpdate(update)
		defer recycle()

		if bot.opts.panicHandler != nil {
			defer func() {
				if e := recover(); e != nil {
					bot.opts.panicHandler(ctx, e)
				}
			}()
		}

		switch {
		case bot.cmdHandlers != nil && ctx.IsCommand():
			bot.commandHandler(ctx)

		default:
			bot.updatesHandler(ctx)
		}
	}
}

func (bot *Bot) handleUpdate(update *tgbotapi.Update) {
	updateHandler := bot.makeUpdateHandler(update)

	if bot.opts.workerPool != nil && !bot.opts.workerPool.IsClosed() {
		if err := bot.opts.workerPool.Submit(updateHandler); err != nil {
			bot.opts.errHandler(err)
		}
		return
	}

	// unlimited number of worker
	if bot.opts.workerNum <= 0 {
		go updateHandler()
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

func (bot *Bot) startWorker() {
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

func (bot *Bot) startWorkers() {
	workNum := bot.opts.workerNum
	if workNum <= 0 {
		workNum = 1
	}

	for i := 0; i < workNum; i++ {
		bot.wg.Add(1)
		go bot.startWorker()
	}
}

func (bot *Bot) startPollUpdates() {
	bot.wg.Add(1)
	go bot.pollUpdates()
}

func (bot *Bot) hijackAPI() *tgbotapi.BotAPI {
	// clone a api and hijack the client.
	api := new(tgbotapi.BotAPI)
	*api = *bot.api
	api.Client = &client{cli: bot.api.Client, ctx: bot.ctx}
	return api
}

func (bot *Bot) pollUpdates() {
	defer func() {
		bot.wg.Done()
		close(bot.updateC)
	}()

	api := bot.hijackAPI()

	for {
		select {
		case <-bot.ctx.Done():
			return

		default:
		}

		updates, err := api.GetUpdates(tgbotapi.UpdateConfig{
			Limit:          bot.opts.limit,
			Offset:         bot.opts.offset,
			Timeout:        bot.opts.updateTimeout,
			AllowedUpdates: bot.opts.allowedUpdates,
		})
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return
			}

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
	bot.startPollUpdates()

	// wait all worker done.
	bot.wg.Wait()

	return nil
}

func (bot *Bot) Stop() context.Context {
	bot.cancel()

	if !bot.opts.disableHandleAllUpdateOnStop {
		// must be processed until all updates are processed.
		for update := range bot.updateC {
			bot.wg.Add(1)
			go func(update *tgbotapi.Update) {
				defer bot.wg.Done()
				bot.makeUpdateHandler(update)()
			}(update)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		bot.wg.Wait()
		cancel()
	}()
	return ctx
}
