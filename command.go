package tgbot

import (
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// CommandScope represent a telegram command scope.
type CommandScope struct {
	invalid bool

	Type   string
	ChatID int64
	UserID int64

	LanguageCode string
}

// Command is telegram command.
type Command struct {
	Name        string
	Description string
	Hide        bool // hide the command on telegram commands menu.
	Scopes      []CommandScope
	Handler     Handler
}

func (c Command) String() string {
	return fmt.Sprintf("/%s - %s", c.Name, c.Description)
}

func (c CommandScope) toScope() tgbotapi.BotCommandScope {
	return tgbotapi.BotCommandScope{
		Type:   c.Type,
		ChatID: c.ChatID,
		UserID: c.UserID,
	}
}
