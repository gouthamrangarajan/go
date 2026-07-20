package models

import (
	"github.com/starfederation/datastar-go/datastar"
)

type SearchSessionViaChannelRequest struct {
	SearchTerm string
	UserId     string
}

type LongSSEData struct {
	Content           string
	IsScript          bool
	IsError           bool
	IsRemove          bool
	IsSignal          bool
	Selector          string
	Mode              datastar.PatchElementOption
	UseViewTransition bool
	SendHeartBeat     bool
}

type SessionChangeData struct {
	Session           ChatSession
	ChatConversations []ChatConversation
	SearchMenuText    string
	UserId            string
}

type ChatConversationMarkdownToHtml struct {
	ConversationId int
	Html           string
}
