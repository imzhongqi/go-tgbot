package tgbot

import (
	"context"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type MessageConfigOption func(c *tgbotapi.MessageConfig)

type Context struct {
	context.Context

	*tgbotapi.BotAPI

	message *tgbotapi.Message
	update  *tgbotapi.Update
}

func (ctx *Context) Command() string {
	return ctx.message.Command()
}

func (ctx *Context) CommandArgs() string {
	return ctx.message.CommandArguments()
}

func (ctx *Context) Message() *tgbotapi.Message {
	return ctx.message
}

func (ctx *Context) Update() *tgbotapi.Update {
	return ctx.update
}

func (ctx *Context) FromChat() *tgbotapi.Chat {
	return ctx.update.FromChat()
}

func (ctx *Context) ReplyText(text string, opts ...MessageConfigOption) error {
	return ctx.reply(text, nil, opts...)
}

func (ctx *Context) ReplyMarkdown(text string, opts ...MessageConfigOption) error {
	return ctx.reply(text, func(c *tgbotapi.MessageConfig) {
		c.ParseMode = tgbotapi.ModeMarkdown
	}, opts...)
}

func (ctx *Context) ReplyHTML(text string, opts ...MessageConfigOption) error {
	return ctx.reply(text, func(c *tgbotapi.MessageConfig) {
		c.ParseMode = tgbotapi.ModeHTML
	}, opts...)
}

func (ctx *Context) reply(text string, dc MessageConfigOption, opts ...MessageConfigOption) error {
	msg := tgbotapi.NewMessage(ctx.update.FromChat().ID, text)
	msg.DisableWebPagePreview = true
	if dc != nil {
		dc(&msg)
	}
	for _, o := range opts {
		o(&msg)
	}
	_, err := ctx.Send(msg)
	return err
}

// WithEnableWebPagePreview enable web page preview
func WithEnableWebPagePreview() MessageConfigOption {
	return func(c *tgbotapi.MessageConfig) {
		c.DisableWebPagePreview = false
	}
}

// WithChatId set message chat id
func WithChatId(chatId int64) MessageConfigOption {
	return func(c *tgbotapi.MessageConfig) {
		c.ChatID = chatId
	}
}
