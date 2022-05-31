package tgbot

import (
	"context"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Context struct {
	context.Context

	*tgbotapi.BotAPI

	bot *Bot

	update *tgbotapi.Update
}

// Command return command name if message is non-nil.
func (c *Context) Command() string {
	if message := c.Message(); message != nil {
		return message.Command()
	}
	return ""
}

// IsCommand report whether the current message is a command.
func (c *Context) IsCommand() bool {
	if msg := c.Message(); msg != nil {
		return msg.IsCommand()
	}
	return false
}

// CommandArgs return command arguments if message is non-nil.
func (c *Context) CommandArgs() string {
	if message := c.Message(); message != nil {
		return message.CommandArguments()
	}
	return ""
}

func (c *Context) Message() *tgbotapi.Message {
	switch {
	case c.update.Message != nil:
		return c.update.Message

	case c.update.EditedMessage != nil:
		return c.update.EditedMessage

	case c.update.CallbackQuery != nil:
		return c.update.CallbackQuery.Message

	case c.update.ChannelPost != nil:
		return c.update.ChannelPost

	case c.update.EditedChannelPost != nil:
		return c.update.EditedChannelPost

	default:
		return nil
	}
}

func (c *Context) Update() *tgbotapi.Update {
	return c.update
}

func (c *Context) SentFrom() *tgbotapi.User {
	return c.update.SentFrom()
}

func (c *Context) FromChat() *tgbotapi.Chat {
	return c.update.FromChat()
}

type MessageOption func(c *tgbotapi.MessageConfig)

// ReplyText reply to the current chat.
func (c *Context) ReplyText(text string, opts ...MessageOption) error {
	return c.reply(text, mergeOpts(opts,
		WithDisableWebPagePreview(true),
	)...)
}

// ReplyMarkdown reply to the current chat, text format is markdown.
func (c *Context) ReplyMarkdown(text string, opts ...MessageOption) error {
	return c.reply(text, mergeOpts(opts,
		WithMarkdown(),
		WithDisableWebPagePreview(true),
	)...)
}

// ReplyHTML reply to the current chat, text format is HTML.
func (c *Context) ReplyHTML(text string, opts ...MessageOption) error {
	return c.reply(text, mergeOpts(opts,
		WithHTML(),
		WithDisableWebPagePreview(true),
	)...)
}

func (c *Context) reply(text string, opts ...MessageOption) error {
	msg := tgbotapi.NewMessage(0, text)
	if chat := c.update.FromChat(); chat != nil {
		msg.ChatID = chat.ID
	}
	for _, o := range opts {
		o(&msg)
	}
	return c.SendReply(msg)
}

// SendReply send reply.
func (c *Context) SendReply(chat tgbotapi.Chattable) error {
	_, err := c.Request(chat)
	return err
}

// WithContext clone a Context for use in other goroutine.
func (c *Context) WithContext(ctx context.Context) *Context {
	nc := c.clone()
	nc.Context = ctx
	if cli, ok := nc.BotAPI.Client.(*client); ok {
		nc.BotAPI.Client = cli.withContext(nc.Context)
	}
	return nc
}

func (c *Context) clone() *Context {
	nc := new(Context)
	*nc = *c
	return nc
}

func (c *Context) reset() {
	c.update = nil
	c.Context = nil
}

func (c *Context) put() {
	c.reset()
	c.bot.pool.Put(c)
}

func mergeOpts(opts []MessageOption, def ...MessageOption) []MessageOption {
	return append(def, opts...)
}

// WithHTML set parse mode to html
func WithHTML() MessageOption {
	return func(c *tgbotapi.MessageConfig) {
		c.ParseMode = tgbotapi.ModeHTML
	}
}

// WithMarkdown set parse mode to markdown.
func WithMarkdown() MessageOption {
	return func(c *tgbotapi.MessageConfig) {
		c.ParseMode = tgbotapi.ModeMarkdown
	}
}

// WithMarkdownV2 set parse mode to markdown v2.
func WithMarkdownV2() MessageOption {
	return func(c *tgbotapi.MessageConfig) {
		c.ParseMode = tgbotapi.ModeMarkdownV2
	}
}

// WithDisableWebPagePreview disable web page preview.
func WithDisableWebPagePreview(disable bool) MessageOption {
	return func(c *tgbotapi.MessageConfig) {
		c.DisableWebPagePreview = disable
	}
}

// WithChatId set message chat id.
func WithChatId(chatId int64) MessageOption {
	return func(c *tgbotapi.MessageConfig) {
		c.ChatID = chatId
	}
}
