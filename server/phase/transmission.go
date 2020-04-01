package phase

import (
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/services"
)

type GetChunk func() (services.Chunk, bool)
type GetMessage func(index uint32) *mixmessages.Slot
type Measure func(tag string)
type Transmit func(roundID id.Round, instance Instance, getChunk GetChunk) error

type Instance interface {
	GetID() *id.Node
	GetNetwork() *node.Comms
}
