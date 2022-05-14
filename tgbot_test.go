package tgbot_test

import (
	"log"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/imzhongqi/tgbot"
)

func ExampleNewBot() {
	api, err := tgbotapi.NewBotAPI("xxx")
	if err != nil {
		panic(err)
	}

	bot := tgbot.NewBot(api,
		tgbot.WithTimeout(2*time.Second),

		tgbot.WithUpdatesHandler(func(ctx *tgbot.Context) {
			err := ctx.ReplyText(ctx.Update().Message.Text, func(c *tgbotapi.MessageConfig) {
				c.ReplyToMessageID = ctx.Message().MessageID
			})
			if err != nil {
				log.Printf("reply text error: %s", err)
			}
		}),

		tgbot.WithUndefinedCmdHandler(func(ctx *tgbot.Context) error {
			return ctx.ReplyMarkdown("*unknown command*", tgbot.WithEnableWebPagePreview())
		}),

		tgbot.WithErrorHandler(func(err error) {
			log.Println(err)
		}),
	)
	bot.AddCommand(&tgbot.Command{
		Name:        "ping",
		Description: "ping the bot",
		Handler: func(ctx *tgbot.Context) error {
			return ctx.ReplyMarkdown("hello,world")
		},
	})
	if err := bot.Run(); err != nil {
		panic(err)
	}
}
