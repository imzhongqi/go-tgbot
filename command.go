package tgbot

import (
	"fmt"
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

var noScope = &commandScope{}

// CommandScope is command scope for telegram.
type CommandScope interface {
	Type() string
	ChatID() int64
	UserID() int64
	LanguageCode() string
}

// commandScope represent a telegram command scope.
type commandScope struct {
	typ    string
	chatID int64
	userID int64

	languageCode string
}

func (c commandScope) Type() string {
	return c.typ
}

func (c commandScope) ChatID() int64 {
	return c.chatID
}

func (c commandScope) UserID() int64 {
	return c.userID
}

func (c commandScope) LanguageCode() string {
	return c.languageCode
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

func CommandScopes(scopes ...CommandScope) []CommandScope {
	return scopes
}

func CommandScopeNoScope() CommandScope {
	return noScope
}

// CommandScopeDefault represents the default scope of bot commands.
func CommandScopeDefault() CommandScope {
	return commandScope{typ: ScopeTypeDefault}
}

// CommandScopeAllPrivateChats represents the scope of bot commands,
// covering all private chats.
func CommandScopeAllPrivateChats() CommandScope {
	return commandScope{typ: ScopeTypeAllPrivateChats}
}

// CommandScopeAllGroupChats represents the scope of bot commands,
// covering all group and supergroup chats.
func CommandScopeAllGroupChats() CommandScope {
	return commandScope{typ: ScopeTypeAllGroupChats}
}

// CommandScopeAllChatAdministrators represents the scope of bot commands,
// covering all group and supergroup chat administrators.
func CommandScopeAllChatAdministrators() CommandScope {
	return commandScope{typ: ScopeTypeAllChatAdministrators}
}

// CommandScopeChat represents the scope of bot commands, covering a
// specific chat.
func CommandScopeChat(chatID int64) CommandScope {
	return commandScope{
		typ:    ScopeTypeChat,
		chatID: chatID,
	}
}

// CommandScopeChatAdministrators represents the scope of bot commands,
// covering all administrators of a specific group or supergroup chat.
func CommandScopeChatAdministrators(chatID int64) CommandScope {
	return commandScope{
		typ:    ScopeTypeChatAdministrators,
		chatID: chatID,
	}
}

// CommandScopeChatMember represents the scope of bot commands, covering a
// specific member of a group or supergroup chat.
func CommandScopeChatMember(chatID, userID int64) CommandScope {
	return commandScope{
		typ:    ScopeTypeChatMember,
		chatID: chatID,
		userID: userID,
	}
}
