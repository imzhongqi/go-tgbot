package tgbot

import (
	"context"
	"errors"
	"fmt"
	"strings"
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

	commands map[string]*Command

	updateC chan *tgbotapi.Update
}

// NewBot new a telegram bot.
func NewBot(api *tgbotapi.BotAPI, opts ...Option) *Bot {
	if api == nil {
		panic("tgbot: api is nil, api must be a non-nil")
	}

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

type multiErr []error

func (e multiErr) Error() string {
	builder := strings.Builder{}
	for _, err := range e {
		builder.WriteString(err.Error())
		builder.WriteByte(' ')
	}
	return builder.String()
}

func (bot *Bot) ClearBotCommands() error {
	wg := sync.WaitGroup{}
	ec := make(chan error)
	request := func(c tgbotapi.Chattable) {
		wg.Add(1)

		go func() {
			defer wg.Done()
			if _, err := bot.api.Request(c); err != nil {
				ec <- err
			}
		}()
	}

	var errs multiErr
	go func() {
		for e := range ec {
			errs = append(errs, e)
		}
	}()

	request(tgbotapi.NewDeleteMyCommands())
	request(tgbotapi.NewDeleteMyCommandsWithScope(tgbotapi.NewBotCommandScopeDefault()))
	request(tgbotapi.NewDeleteMyCommandsWithScope(tgbotapi.NewBotCommandScopeAllPrivateChats()))
	request(tgbotapi.NewDeleteMyCommandsWithScope(tgbotapi.NewBotCommandScopeAllGroupChats()))
	request(tgbotapi.NewDeleteMyCommandsWithScope(tgbotapi.NewBotCommandScopeAllChatAdministrators()))

	wg.Wait()

	close(ec)

	if errs != nil {
		return errs
	}
	return nil
}

// AddCommands add commands to the bot.
func (bot *Bot) AddCommands(commands ...*Command) {
	if bot.commands == nil {
		bot.commands = make(map[string]*Command)
	}

	for _, c := range commands {
		switch {
		case c.Name == "":
			panic("tgbot: command name must be non-empty")
		case c.Description == "":
			panic("tgbot: command description must be non-empty")
		case c.Handler == nil:
			panic("tgbot: command handler must be non-nil")
		}

		if _, ok := bot.commands[c.Name]; ok {
			panic("duplicate command name: " + c.Name)
		}

		bot.commands[c.Name] = c
	}
}

func (bot *Bot) Commands() map[string]*Command {
	return bot.commands
}

func (bot *Bot) CommandsWithScope() map[CommandScope][]*Command {
	commandGroups := make(map[CommandScope][]*Command)
	for _, cmd := range bot.commands {

		// process no scope command.
		if len(cmd.scopes) == 0 {
			commandGroups[noScope] = append(commandGroups[noScope], cmd)
			continue
		}

		for _, scope := range cmd.scopes {
			commandGroups[scope] = append(commandGroups[scope], cmd)
		}
	}
	return commandGroups
}

func (bot *Bot) setupCommands() error {
	if bot.opts.disableAutoSetupCommands {
		return nil
	}

	for scope, commands := range bot.CommandsWithScope() {
		botCommands := make([]tgbotapi.BotCommand, 0, len(commands))
		for _, cmd := range commands {
			if cmd.hide {
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
		case bot.commands != nil && ctx.IsCommand():
			bot.commandHandler(ctx)

		default:
			bot.updatesHandler(ctx)
		}
	}
}

func (bot *Bot) handleUpdate(update *tgbotapi.Update) {
	updateHandler := bot.makeUpdateHandler(update)

	if bot.opts.workersPool != nil && !bot.opts.workersPool.IsClosed() {
		if err := bot.opts.workersPool.Submit(updateHandler); err != nil {
			bot.opts.errHandler(err)
		}
		return
	}

	// unlimited number of workers.
	if bot.opts.workersNum <= 0 {
		go updateHandler()
		return
	}

	updateHandler()
}

func (bot *Bot) commandHandler(ctx *Context) {
	handler := bot.undefinedCmdHandler

	if cmd, ok := bot.commands[ctx.Command()]; ok {
		handler = cmd.Handler
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
	workNum := bot.opts.workersNum
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
