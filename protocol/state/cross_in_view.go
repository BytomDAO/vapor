package state

import (
	"github.com/vapor/protocol/bc"
)

type CrossInViewpoint struct {
	Entries map[bc.Hash]bool
}

func NewCrossInViewpoint() *CrossInViewpoint {
	return &CrossInViewpoint{
		Entries: make(map[bc.Hash]bool),
	}
}
