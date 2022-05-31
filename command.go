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

// NewCommandScopeDefault represents the default scope of bot commands.
func NewCommandScopeDefault() CommandScope {
	return CommandScope{Type: "default"}
}

// NewCommandScopeAllPrivateChats represents the scope of bot commands,
// covering all private chats.
func NewCommandScopeAllPrivateChats() CommandScope {
	return CommandScope{Type: "all_private_chats"}
}

// NewCommandScopeAllGroupChats represents the scope of bot commands,
// covering all group and supergroup chats.
func NewCommandScopeAllGroupChats() CommandScope {
	return CommandScope{Type: "all_group_chats"}
}

// NewCommandScopeAllChatAdministrators represents the scope of bot commands,
// covering all group and supergroup chat administrators.
func NewCommandScopeAllChatAdministrators() CommandScope {
	return CommandScope{Type: "all_chat_administrators"}
}

// NewCommandScopeChat represents the scope of bot commands, covering a
// specific chat.
func NewCommandScopeChat(chatID int64) CommandScope {
	return CommandScope{
		Type:   "chat",
		ChatID: chatID,
	}
}

// NewCommandScopeChatAdministrators represents the scope of bot commands,
// covering all administrators of a specific group or supergroup chat.
func NewCommandScopeChatAdministrators(chatID int64) CommandScope {
	return CommandScope{
		Type:   "chat_administrators",
		ChatID: chatID,
	}
}

// NewCommandScopeChatMember represents the scope of bot commands, covering a
// specific member of a group or supergroup chat.
func NewCommandScopeChatMember(chatID, userID int64) CommandScope {
	return CommandScope{
		Type:   "chat_member",
		ChatID: chatID,
		UserID: userID,
	}
}
