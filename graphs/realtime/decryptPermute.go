package realtime

import (
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/graphs"
	"gitlab.com/elixxir/server/server"
	"gitlab.com/elixxir/server/server/round"
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
func (dps *DecryptPermuteStream) Link(grp *cyclic.Group, batchSize uint32, source ...interface{}) {
	roundBuf := source[0].(*round.Buffer)
	userRegistry := source[1].(*server.Instance).GetUserRegistry()

	users := make([]*id.User, batchSize)

	for i := uint32(0); i < batchSize; i++ {
		users[i] = &id.User{}
	}

	ecrMsg := grp.NewIntBuffer(batchSize, grp.NewInt(1))
	ecrAD := grp.NewIntBuffer(batchSize, grp.NewInt(1))

	dps.LinkRealtimeDecryptStream(grp, batchSize, roundBuf, userRegistry,
		ecrMsg, ecrAD,
		grp.NewIntBuffer(batchSize, grp.NewInt(1)),
		grp.NewIntBuffer(batchSize, grp.NewInt(1)),
		users, make([][]byte, batchSize))

	dps.LinkRealtimePermuteStreams(grp, batchSize, roundBuf, ecrMsg, ecrAD,
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
	gPermute := graphs.ModifyGraphGeneratorForPermute(gc)
	g := gPermute.NewGraph("RealtimeDecryptPermute", &DecryptPermuteStream{})

	decryptKeygen := graphs.Keygen.DeepCopy()
	decryptMul3 := DecryptMul3.DeepCopy()
	permuteMul2 := PermuteMul2.DeepCopy()

	g.First(decryptKeygen)
	g.Connect(decryptKeygen, decryptMul3)
	g.Connect(decryptMul3, permuteMul2)
	g.Last(permuteMul2)

	return g
}
