package tgbot

import (
	"fmt"
)

const (
	ScopeTypeDefault               = "default"
	ScopeTypeAllPrivateChats       = "all_private_chats"
	ScopeTypeAllGroupChats         = "all_group_chats"
	ScopeTypeAllChatAdministrators = "all_chat_administrators"
	ScopeTypeChat                  = "chat"
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
	Handler     Handler

	hide   bool // hide the command on telegram commands menu.
	scopes []CommandScope
}

type CommandOption func(cmd *Command)

func WithHide(v bool) CommandOption {
	return func(cmd *Command) {
		cmd.hide = v
	}
}

func WithScopes(scopes ...CommandScope) CommandOption {
	return func(cmd *Command) {
		cmd.scopes = make([]CommandScope, 0, len(scopes))
		scopeSet := make(map[CommandScope]struct{}, len(scopes))
		for _, scope := range scopes {
			if _, ok := scopeSet[scope]; ok {
				continue
			}
			scopeSet[scope] = struct{}{}
			cmd.scopes = append(cmd.scopes, scope)
		}
	}
}

func NewCommand(name, desc string, handler Handler, opts ...CommandOption) *Command {
	cmd := &Command{
		Name:        name,
		Description: desc,
		Handler:     handler,
	}
	for _, opt := range opts {
		opt(cmd)
	}
	return cmd
}

func (c Command) String() string {
	return fmt.Sprintf("/%s - %s", c.Name, c.Description)
}

func (c *Command) Hide() bool {
	return c.hide
}

func (c *Command) Scopes() []CommandScope {
	return c.scopes
}

func CommandScopeNoScope() CommandScope {
	return noScope
}

func lang(lc ...string) string {
	if len(lc) > 0 {
		return lc[0]
	}
	return ""
}

// CommandScopeDefault represents the default scope of bot commands.
func CommandScopeDefault(lc ...string) CommandScope {
	return commandScope{typ: ScopeTypeDefault, languageCode: lang(lc...)}
}

// CommandScopeAllPrivateChats represents the scope of bot commands,
// covering all private chats.
func CommandScopeAllPrivateChats(lc ...string) CommandScope {
	return commandScope{typ: ScopeTypeAllPrivateChats, languageCode: lang(lc...)}
}

// CommandScopeAllGroupChats represents the scope of bot commands,
// covering all group and supergroup chats.
func CommandScopeAllGroupChats(lc ...string) CommandScope {
	return commandScope{typ: ScopeTypeAllGroupChats, languageCode: lang(lc...)}
}

// CommandScopeAllChatAdministrators represents the scope of bot commands,
// covering all group and supergroup chat administrators.
func CommandScopeAllChatAdministrators(lc ...string) CommandScope {
	return commandScope{typ: ScopeTypeAllChatAdministrators, languageCode: lang(lc...)}
}

// CommandScopeChat represents the scope of bot commands, covering a
// specific chat.
func CommandScopeChat(chatID int64, lc ...string) CommandScope {
	return commandScope{
		typ:          ScopeTypeChat,
		chatID:       chatID,
		languageCode: lang(lc...),
	}
}

// CommandScopeChatAdministrators represents the scope of bot commands,
// covering all administrators of a specific group or supergroup chat.
func CommandScopeChatAdministrators(chatID int64, lc ...string) CommandScope {
	return commandScope{
		typ:          ScopeTypeChatAdministrators,
		chatID:       chatID,
		languageCode: lang(lc...),
	}
}

// CommandScopeChatMember represents the scope of bot commands, covering a
// specific member of a group or supergroup chat.
func CommandScopeChatMember(chatID, userID int64, lc ...string) CommandScope {
	return commandScope{
		typ:          ScopeTypeChatMember,
		chatID:       chatID,
		userID:       userID,
		languageCode: lang(lc...),
	}
}
