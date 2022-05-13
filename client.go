package tgbot

import (
	"context"
	"net/http"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type client struct {
	cli tgbotapi.HTTPClient
	ctx context.Context
}

func (c *client) Do(req *http.Request) (*http.Response, error) {
	return c.cli.Do(req.WithContext(c.ctx))
}
