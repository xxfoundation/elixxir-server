package server

import (
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/server/node"
	"gitlab.com/elixxir/server/services"
)

type GetChunk func() (services.Chunk, bool)
type GetMessage func(index uint32) *mixmessages.CmixSlot
type Transmission func(round *Round, phase node.PhaseType,
	getSlot GetChunk, getMessage GetMessage)
