package phase

import (
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/services"
)

type GetChunk func() (services.Chunk, bool)
type GetMessage func(index uint32) *mixmessages.Slot
type Measure func(tag string)

// Fixme: getmessage can be removed from the interface, but it makes testing difficult.
//  A more general refactor is required to remove this while keeping testability
type Transmit func(roundID id.Round, instance GenericInstance, getChunk GetChunk, getMessage GetMessage) error

type GenericInstance interface {
}
