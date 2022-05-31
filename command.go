package tgbot

import (
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	ScopeTypeDefault               = "default"
	ScopeTypeAllPrivateChats       = "all_private_chats"
	ScopeTypeAllGroupChats         = "all_group_chats"
	ScopeTypeAllChatAdministrators = "all_chat_administrators"
	ScopeTypeChat                  = "all_chat_administrators"
	ScopeTypeChatAdministrators    = "chat_administrators"
	ScopeTypeChatMember            = "chat_member"
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
	return CommandScope{Type: ScopeTypeDefault}
}

// NewCommandScopeAllPrivateChats represents the scope of bot commands,
// covering all private chats.
func NewCommandScopeAllPrivateChats() CommandScope {
	return CommandScope{Type: ScopeTypeAllPrivateChats}
}

// NewCommandScopeAllGroupChats represents the scope of bot commands,
// covering all group and supergroup chats.
func NewCommandScopeAllGroupChats() CommandScope {
	return CommandScope{Type: ScopeTypeAllGroupChats}
}

// NewCommandScopeAllChatAdministrators represents the scope of bot commands,
// covering all group and supergroup chat administrators.
func NewCommandScopeAllChatAdministrators() CommandScope {
	return CommandScope{Type: ScopeTypeAllChatAdministrators}
}

// NewCommandScopeChat represents the scope of bot commands, covering a
// specific chat.
func NewCommandScopeChat(chatID int64) CommandScope {
	return CommandScope{
		Type:   ScopeTypeChat,
		ChatID: chatID,
	}
}

// NewCommandScopeChatAdministrators represents the scope of bot commands,
// covering all administrators of a specific group or supergroup chat.
func NewCommandScopeChatAdministrators(chatID int64) CommandScope {
	return CommandScope{
		Type:   ScopeTypeChatAdministrators,
		ChatID: chatID,
	}
}

// NewCommandScopeChatMember represents the scope of bot commands, covering a
// specific member of a group or supergroup chat.
func NewCommandScopeChatMember(chatID, userID int64) CommandScope {
	return CommandScope{
		Type:   ScopeTypeChatMember,
		ChatID: chatID,
		UserID: userID,
	}
}
