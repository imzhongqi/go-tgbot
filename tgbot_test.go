package tgbot_test

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/imzhongqi/tgbot"
)

func ExampleNewBot() {
	api, err := tgbotapi.NewBotAPI("xxx")
	if err != nil {
		panic(err)
	}

	bot := tgbot.NewBot(api)
	bot.AddCommand(&tgbot.Command{
		Name:        "ping",
		Description: "ping the bot",
		Handler: func(api *tgbotapi.BotAPI, message *tgbotapi.Message) error {
			return tgbot.NewBotAPI(api).ReplyMarkdown(message, "hello,world")
		},
	})
	if err := bot.Run(); err != nil {
		panic(err)
	}

	//output:

}
