package phase

import (
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/services"
)

type GetChunk func() (services.Chunk, bool)
type GetMessage func(index uint32) *mixmessages.Slot

type Transmit func(batchSize uint32, roundID id.Round, phaseTy Type, getChunk GetChunk,
	getMessage GetMessage, nal *services.NodeAddressList) error
