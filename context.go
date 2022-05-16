package tgbot

import (
	"context"
	"errors"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type MessageConfigOption func(c *tgbotapi.MessageConfig)

type Context struct {
	context.Context

	*tgbotapi.BotAPI

	bot *Bot

	update *tgbotapi.Update
}

// Command return command name if message is non-nil
func (ctx *Context) Command() string {
	if message := ctx.Message(); message != nil {
		return message.Command()
	}
	return ""
}

// IsCommand report whether the current message is a command
func (ctx *Context) IsCommand() bool {
	if msg := ctx.Message(); msg != nil {
		return msg.IsCommand()
	}
	return false
}

// CommandArgs return command arguments if message is non-nil
func (ctx *Context) CommandArgs() string {
	if message := ctx.Message(); message != nil {
		return message.CommandArguments()
	}
	return ""
}

func (ctx *Context) Message() *tgbotapi.Message {
	switch {
	case ctx.update.Message != nil:
		return ctx.update.Message
	case ctx.update.EditedMessage != nil:
		return ctx.update.EditedMessage
	case ctx.update.ChannelPost != nil:
		return ctx.update.ChannelPost
	case ctx.update.EditedChannelPost != nil:
		return ctx.update.EditedChannelPost
	default:
		return nil
	}
}

func (ctx *Context) Update() *tgbotapi.Update {
	return ctx.update
}

func (ctx *Context) SentFrom() *tgbotapi.User {
	return ctx.update.SentFrom()
}

func (ctx *Context) FromChat() *tgbotapi.Chat {
	return ctx.update.FromChat()
}

// ReplyText reply to the current chat
func (ctx *Context) ReplyText(text string, opts ...MessageConfigOption) error {
	return ctx.reply(text, opts...)
}

// ReplyMarkdown reply to the current chat, text format is markdown
func (ctx *Context) ReplyMarkdown(text string, opts ...MessageConfigOption) error {
	return ctx.reply(text, mergeOpts(opts,
		WithMarkdown(),
		WithDisableWebPagePreview(true),
	)...)
}

// ReplyHTML reply to the current chat, text format is HTML
func (ctx *Context) ReplyHTML(text string, opts ...MessageConfigOption) error {
	return ctx.reply(text, mergeOpts(opts,
		WithHTML(),
		WithDisableWebPagePreview(true),
	)...)
}

func (ctx *Context) SendReply(chat tgbotapi.Chattable) error {
	_, err := ctx.Request(chat)
	return err
}

func (ctx *Context) reply(text string, opts ...MessageConfigOption) error {
	chat := ctx.update.FromChat()
	if chat == nil {
		return errors.New("no chat currently")
	}
	msg := tgbotapi.NewMessage(chat.ID, text)
	for _, o := range opts {
		o(&msg)
	}
	return ctx.SendReply(msg)
}

func (ctx *Context) WithContext(c context.Context) *Context {
	nc := ctx.clone()
	nc.Context = c
	return nc
}

func (ctx *Context) clone() *Context {
	nc := new(Context)
	*nc = *ctx
	return nc
}

func (ctx *Context) reset() {
	ctx.update = nil
	ctx.Context = nil
}

func (ctx *Context) put() {
	ctx.reset()
	ctx.bot.pool.Put(ctx)
}

func mergeOpts(opts []MessageConfigOption, def ...MessageConfigOption) []MessageConfigOption {
	return append(def, opts...)
}

// WithHTML set parse mode to html
func WithHTML() MessageConfigOption {
	return func(c *tgbotapi.MessageConfig) {
		c.ParseMode = tgbotapi.ModeHTML
	}
}

// WithMarkdown set parse mode to markdown
func WithMarkdown() MessageConfigOption {
	return func(c *tgbotapi.MessageConfig) {
		c.ParseMode = tgbotapi.ModeMarkdown
	}
}

// WithMarkdownV2 set parse mode to markdown v2
func WithMarkdownV2() MessageConfigOption {
	return func(c *tgbotapi.MessageConfig) {
		c.ParseMode = tgbotapi.ModeMarkdownV2
	}
}

// WithDisableWebPagePreview disable web page preview
func WithDisableWebPagePreview(disable bool) MessageConfigOption {
	return func(c *tgbotapi.MessageConfig) {
		c.DisableWebPagePreview = disable
	}
}

// WithChatId set message chat id
func WithChatId(chatId int64) MessageConfigOption {
	return func(c *tgbotapi.MessageConfig) {
		c.ChatID = chatId
	}
}
