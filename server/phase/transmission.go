package phase

import (
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/primitives/circuit"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/services"
)

type GetChunk func() (services.Chunk, bool)
type GetMessage func(index uint32) *mixmessages.Slot

type Transmit func(network *node.NodeComms, batchSize uint32,
	roundID id.Round, phaseTy Type, getChunk GetChunk,
	getMessage GetMessage, topology *circuit.Circuit, nodeId *id.Node) error
