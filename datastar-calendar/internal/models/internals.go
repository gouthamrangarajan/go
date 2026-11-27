package models

import "github.com/starfederation/datastar-go/datastar"

type LongSSEData struct {
	Content           string
	Selector          string
	Mode              datastar.PatchElementOption
	UseViewTransition bool
	IsRemove          bool
	IsSignals         bool
	IsScript          bool
}
