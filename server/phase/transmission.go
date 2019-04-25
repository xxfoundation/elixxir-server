package phase

import (
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/server/services"
)

type GetChunk func() (services.Chunk, bool)
type GetMessage func(index uint32) *mixmessages.Slot
type Transmission func(phase *Phase, nal *services.NodeAddressList,
	getSlot GetChunk, getMessage GetMessage)
