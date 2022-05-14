# tgbot
[![Go Reference](https://pkg.go.dev/badge/github.com/imzhongqi/tgbot.svg)](https://pkg.go.dev/github.com/imzhongqi/tgbot)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Report Card](https://goreportcard.com/badge/github.com/imzhongqi/tgbot)](https://goreportcard.com/report/github.com/imzhongqi/tgbot)

telegram Bot, 包装了一层 [telegram-bot-api](https://github.com/go-telegram-bot-api/telegram-bot-api), 为了更加便捷的创建 telegram 机器人， 提供了一个简单便捷的 API。

```go
package main

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/imzhongqi/tgbot"
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
```

