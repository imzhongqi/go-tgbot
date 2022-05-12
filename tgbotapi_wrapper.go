package tgbot

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type BotAPI struct {
	*tgbotapi.BotAPI
}

func NewBotAPI(api *tgbotapi.BotAPI) *BotAPI {
	return &BotAPI{api}
}

func (a *BotAPI) ReplyText(message *tgbotapi.Message, text string, opts ...MessageConfigOption) error {
	_, err := a.Send(NewMessage(message.Chat.ID, text, opts...))
	return err
}

func (a *BotAPI) ReplyMarkdown(message *tgbotapi.Message, text string, opts ...MessageConfigOption) error {
	msg := NewMessage(message.Chat.ID, text, opts...)
	msg.ParseMode = tgbotapi.ModeMarkdown
	msg.DisableWebPagePreview = true
	_, err := a.Send(msg)
	return err
}

type MessageConfigOption = func(c *tgbotapi.MessageConfig)

func NewMessage(chatId int64, text string, opts ...MessageConfigOption) tgbotapi.MessageConfig {
	msg := tgbotapi.NewMessage(chatId, text)
	for _, o := range opts {
		o(&msg)
	}
	return msg
}
