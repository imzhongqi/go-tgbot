package tgbot

import (
	"context"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type MessageConfigOption = func(c *tgbotapi.MessageConfig)

func NewMessage(chatId int64, text string, opts ...MessageConfigOption) tgbotapi.MessageConfig {
	msg := tgbotapi.NewMessage(chatId, text)
	for _, o := range opts {
		o(&msg)
	}
	return msg
}

type Context struct {
	context.Context

	*tgbotapi.BotAPI

	Message *tgbotapi.Message

	Update *tgbotapi.Update
}

func (ctx *Context) ReplyText(text string, opts ...MessageConfigOption) error {
	_, err := ctx.Send(NewMessage(ctx.Message.Chat.ID, text, opts...))
	return err
}

func (ctx *Context) ReplyMarkdown(text string, opts ...MessageConfigOption) error {
	msg := NewMessage(ctx.Message.Chat.ID, text, opts...)
	msg.ParseMode = tgbotapi.ModeMarkdown
	msg.DisableWebPagePreview = true
	_, err := ctx.Send(msg)
	return err
}
