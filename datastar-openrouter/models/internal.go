package models

import "github.com/starfederation/datastar-go/datastar"

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
}
