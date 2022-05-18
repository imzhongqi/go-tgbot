# go-tgbot

[![Go Reference](https://pkg.go.dev/badge/github.com/imzhongqi/go-tgbot.svg)](https://pkg.go.dev/github.com/imzhongqi/go-tgbot)
[![Go Report Card](https://goreportcard.com/badge/github.com/imzhongqi/go-tgbot)](https://goreportcard.com/report/github.com/imzhongqi/go-tgbot)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

Wrapped [telegram-bot-api](https://github.com/go-telegram-bot-api/telegram-bot-api) to create telegram bot faster.

## 1. Installation

Run the following command under your project:

```
go get -u github.com/imzhongqi/go-tgbot
```

## 2. Example

```go
package main

import (
	"log"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/imzhongqi/go-tgbot"
	"github.com/panjf2000/ants/v2"
)

func main() {
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
		
		tgbot.WithWorkerPool(pool),

		tgbot.WithUpdatesHandler(func(ctx *tgbot.Context) {
			err := ctx.ReplyText(ctx.Update().Message.Text, func(c *tgbotapi.MessageConfig) {
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
	bot.AddCommands(&tgbot.Command{
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
```

