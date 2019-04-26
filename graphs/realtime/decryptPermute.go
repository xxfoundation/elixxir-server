package realtime

import (
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/graphs"
	"gitlab.com/elixxir/server/node"
	"gitlab.com/elixxir/server/services"
)

type DecryptPermuteStream struct {
	DecryptStream
	PermuteStream
}

// GetName returns the name of the stream for debugging purposes.
func (dps *DecryptPermuteStream) GetName() string {
	return "RealtimeDecryptPermuteStream"
}

// Link binds the two embedded streams together and to the round.
func (dps *DecryptPermuteStream) Link(batchSize uint32, source interface{}) {
	round := source.(*node.RoundBuffer)

	users := make([]*id.User, batchSize)

	for i := uint32(0); i < batchSize; i++ {
		users[i] = &id.User{}
	}

	ecrMsg := round.Grp.NewIntBuffer(batchSize, round.Grp.NewInt(1))
	ecrAD := round.Grp.NewIntBuffer(batchSize, round.Grp.NewInt(1))

	dps.LinkRealtimeDecryptStream(batchSize, round, ecrMsg, ecrAD,
		round.Grp.NewIntBuffer(batchSize, round.Grp.NewInt(1)),
		round.Grp.NewIntBuffer(batchSize, round.Grp.NewInt(1)),
		users, make([][]byte, batchSize))

	dps.LinkRealtimePermuteStreams(batchSize, round, ecrMsg, ecrAD,
		make([]*cyclic.Int, batchSize),
		make([]*cyclic.Int, batchSize))
}

// Input initializes stream inputs from slot.
func (dps *DecryptPermuteStream) Input(index uint32, slot *mixmessages.Slot) error {
	return dps.DecryptStream.Input(index, slot)
}

// Output returns a message with the stream data.
func (dps *DecryptPermuteStream) Output(index uint32) *mixmessages.Slot {
	return dps.PermuteStream.Output(index)
}

// InitDecryptPermuteGraph initializes and returns a new graph.
func InitDecryptPermuteGraph(gc services.GraphGenerator) *services.Graph {
	g := gc.NewGraph("RealtimeDecryptPermute", &DecryptPermuteStream{})

	decryptKeygen := graphs.Keygen.DeepCopy()
	decryptMul3 := DecryptMul3.DeepCopy()
	permuteMul2 := PermuteMul2.DeepCopy()
	permutePerm := graphs.Permute.DeepCopy()

	g.First(decryptKeygen)
	g.Connect(decryptKeygen, decryptMul3)
	g.Connect(decryptMul3, permuteMul2)
	g.Connect(permuteMul2, permutePerm)
	g.Last(permutePerm)

	return g
}
