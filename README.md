# tgbot

telegram Bot, 包装了一层 [telegram-bot-api](https://github.com/go-telegram-bot-api/telegram-bot-api), 为了更加便捷的创建 telegram 机器人， 提供了一个简单便捷的 API。

```go
package main

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/imzhongqi/tgbot"
)

func main() {
	api, err := tgbotapi.NewBotAPI("xxx")
	if err != nil {
		panic(err)
	}

	bot := tgbot.NewBot(api)
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

