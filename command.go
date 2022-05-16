package tgbot

import (
	"fmt"
)

type CommandScope struct {
	Type   string
	ChatID int64
	UserID int64
}

// Command is telegram command
type Command struct {
	Name        string
	Description string
	Hide        bool // hide the command on telegram commands menu
	Scopes      []CommandScope
	Handler     Handler
}

func (c Command) String() string {
	return fmt.Sprintf("/%s - %s", c.Name, c.Description)
}
