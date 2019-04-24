package server

import (
	"gitlab.com/elixxir/server/node"
	"sync"
)

//fixme: make high level system managment
var rm RoundManager

type RoundManager sync.Map

func GetRoundManager() *RoundManager {
	return &rm
}

//fixme: write initializer

func (rm *RoundManager) GetRound(id node.RoundID) *Round {
	r, ok := (*sync.Map)(rm).Load(id)

	if !ok {
		return nil
	}

	return r.(*Round)
}
