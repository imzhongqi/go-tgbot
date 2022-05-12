package tgbot

import (
	"fmt"
	"log"
	"runtime"
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

// Handler command handler
type Handler func(api *tgbotapi.BotAPI, message *tgbotapi.Message) error

// ErrHandler error handler
type ErrHandler func(err error)

// Bot wrapper the telegram bot
type Bot struct {
	api *tgbotapi.BotAPI

	commands    []*Command
	cmdHandlers map[string]Handler

	undefinedCommandHandler Handler
	errHandler              ErrHandler

	workerNum  int
	workerPool *ants.Pool

	timeout int

	bufSize int
	limit   int
	offset  int

	closeC  chan struct{}
	updateC chan tgbotapi.Update
}

type Option func(b *Bot)

// WithTimeout set the get updates timeout.
func WithTimeout(timeout int) Option {
	return func(b *Bot) {
		b.timeout = timeout
	}
}

// WithWorkerNum set the number of workers to process updates.
func WithWorkerNum(n int) Option {
	return func(b *Bot) {
		if b.workerNum > 0 {
			b.workerNum = n
		}
	}
}

// WithWorkerPool set the worker pool for execute handler if the workerPool is non-nil.
func WithWorkerPool(p *ants.Pool) Option {
	return func(b *Bot) {
		b.workerPool = p
	}
}

// WithUndefinedCmdHandler set how to handle undefined commands.
func WithUndefinedCmdHandler(h Handler) Option {
	return func(b *Bot) {
		if h != nil {
			b.undefinedCommandHandler = h
		}
	}
}

// WithErrorHandler set error handler
func WithErrorHandler(h ErrHandler) Option {
	return func(b *Bot) {
		if h != nil {
			b.errHandler = h
		}
	}
}

// WithBufferSize set the buffer size for receive updates.
func WithBufferSize(size int) Option {
	return func(b *Bot) {
		b.bufSize = size
	}
}

// WithLimitUpdates set the get updates limit.
func WithLimitUpdates(limit int) Option {
	return func(b *Bot) {
		b.limit = limit
	}
}

func NewBot(api *tgbotapi.BotAPI, opts ...Option) *Bot {
	bot := &Bot{
		api:         api,
		cmdHandlers: make(map[string]Handler),
		timeout:     60,
		errHandler:  func(err error) {},
		workerNum:   runtime.GOMAXPROCS(0),
		closeC:      make(chan struct{}),
		limit:       100,
	}

	for _, o := range opts {
		o(bot)
	}

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
	commands := make([]tgbotapi.BotCommand, 0, len(bot.commands)+1)
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
	// only handler message
	if update.Message == nil || !update.Message.IsCommand() {
		return
	}

	defer func() {
		if e := recover(); e != nil {
			bot.errHandler(fmt.Errorf("handlerUpdate recover: %v", e))

			var chatId int64
			switch {
			case update.Message != nil:
				chatId = update.Message.Chat.ID

			default:
				return
			}

			if _, err := bot.api.Send(tgbotapi.NewMessage(chatId, "oops! Service is temporarily unavailable")); err != nil {
				bot.errHandler(err)
			}
		}
	}()

	handler, ok := bot.cmdHandlers[update.Message.Command()]
	if !ok {
		handler = bot.undefinedCmdHandler
	}

	// use workerPool if workerPool available
	if bot.workerPool != nil {
		if err := bot.workerPool.Submit(func() {
			if err := handler(bot.api, update.Message); err != nil {
				bot.errHandler(err)
			}
		}); err != nil {
			bot.errHandler(err)
		}
		return
	}

	if err := handler(bot.api, update.Message); err != nil {
		bot.errHandler(err)
	}
}

func (bot *Bot) undefinedCmdHandler(api *tgbotapi.BotAPI, message *tgbotapi.Message) error {
	if bot.undefinedCommandHandler != nil {
		return bot.undefinedCommandHandler(api, message)
	}
	return NewBotAPI(api).ReplyText(message, "Unrecognized command!!!")
}

func (bot *Bot) startWorker() {
	startWorker := func() {
		for {
			select {
			case <-bot.closeC:
				return

			case update := <-bot.updateC:
				bot.handleUpdate(update)
			}
		}
	}

	for i := 0; i < bot.workerNum; i++ {
		go startWorker()
	}
}

func (bot *Bot) pollUpdates() {
	for {
		select {
		case <-bot.closeC:
			return
		default:
		}

		updates, err := bot.api.GetUpdates(tgbotapi.UpdateConfig{
			Limit:   bot.limit,
			Offset:  bot.offset,
			Timeout: bot.timeout,
		})
		if err != nil {
			bot.errHandler(err)
			log.Println("Failed to get updates, retrying in 3 seconds...")
			time.Sleep(time.Second * 3)
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

	return nil
}

func (bot *Bot) Stop() {
	close(bot.closeC)
}
