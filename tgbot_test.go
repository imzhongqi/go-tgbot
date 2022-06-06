package tgbot_test

import (
	"log"
	"testing"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/imzhongqi/go-tgbot"
	"github.com/panjf2000/ants/v2"
)

func ExampleNewBot() {
	api, err := tgbotapi.NewBotAPI("xxx")
	if err != nil {
		panic(err)
	}

	pool, err := ants.NewPool(10000, ants.WithExpiryDuration(10*time.Second))
	if err != nil {
		panic(err)
	}

	bot := tgbot.NewBot(api,
		tgbot.WithTimeout(2*time.Second),

		tgbot.WithWorkersPool(pool),

		tgbot.WithUpdatesHandler(func(ctx *tgbot.Context) {
			err := ctx.ReplyText(ctx.Message().Text, func(c *tgbotapi.MessageConfig) {
				c.ReplyToMessageID = ctx.Message().MessageID
			})
			if err != nil {
				log.Printf("reply text error: %s", err)
			}
		}),

		tgbot.WithUndefinedCmdHandler(func(ctx *tgbot.Context) error {
			return ctx.ReplyMarkdown("*unknown command*", tgbot.WithDisableWebPagePreview(false))
		}),

		tgbot.WithErrorHandler(func(err error) {
			log.Println(err)
		}),
	)
	bot.AddCommands(
		tgbot.NewCommand("ping", "ping the bot", func(ctx *tgbot.Context) error {
			return ctx.ReplyMarkdown("pong")
		},
			tgbot.WithHide(true),
			tgbot.WithScopes(
				tgbot.CommandScopeDefault(),
				tgbot.CommandScopeAllGroupChats(),
				tgbot.CommandScopeChat(100),
			),
		),
	)
	if err := bot.Run(); err != nil {
		panic(err)
	}
}

func TestNewBot(t *testing.T) {
	defer func() {
		if e := recover(); e == nil {
			t.Error("must be trigger panic")
		}
	}()
	tgbot.NewBot(nil)
}
