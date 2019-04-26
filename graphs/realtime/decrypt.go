////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package realtime

import (
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/cryptops"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/globals"
	"gitlab.com/elixxir/server/graphs"
	"gitlab.com/elixxir/server/node"
	"gitlab.com/elixxir/server/services"
)

// Stream holding data containing keys and inputs used by decrypt
type DecryptStream struct {
	Grp *cyclic.Group

	// Link to round object
	R *cyclic.IntBuffer
	U *cyclic.IntBuffer

	// Unique to stream
	EcrMsg *cyclic.IntBuffer
	EcrAD  *cyclic.IntBuffer

	// Components for key generation
	Users   []*id.User
	Salts   [][]byte
	KeysMsg *cyclic.IntBuffer
	KeysAD  *cyclic.IntBuffer

	graphs.KeygenSubStream
}

func (s *DecryptStream) GetName() string {
	return "RealtimeDecryptStream"
}

//Link creates the stream's internal buffers and links to the round
func (ds *DecryptStream) Link(batchSize uint32, source interface{}) {
	round := source.(*node.RoundBuffer)

	users := make([]*id.User, batchSize)

	for i := uint32(0); i < batchSize; i++ {
		users[i] = &id.User{}
	}

	ds.LinkRealtimeDecryptStream(batchSize, round,
		round.Grp.NewIntBuffer(batchSize, round.Grp.NewInt(1)),
		round.Grp.NewIntBuffer(batchSize, round.Grp.NewInt(1)),
		round.Grp.NewIntBuffer(batchSize, round.Grp.NewInt(1)),
		round.Grp.NewIntBuffer(batchSize, round.Grp.NewInt(1)),
		users, make([][]byte, batchSize))
}

//Connects the internal buffers in the stream to the passed
func (ds *DecryptStream) LinkRealtimeDecryptStream(batchSize uint32, round *node.RoundBuffer,
	ecrMsg, ecrAD, keysMsg, keysAD *cyclic.IntBuffer, users []*id.User, salts [][]byte) {

	ds.Grp = round.Grp

	ds.R = round.R.GetSubBuffer(0, batchSize)
	ds.U = round.U.GetSubBuffer(0, batchSize)

	ds.EcrMsg = ecrMsg
	ds.EcrAD = ecrAD
	ds.KeysMsg = keysMsg
	ds.KeysAD = keysAD
	ds.Users = users
	ds.Salts = salts

	ds.KeygenSubStream.LinkStream(ds.Grp, ds.Salts, ds.Users, ds.KeysMsg, ds.KeysAD)
}

// PermuteStream conforms to this interface.
type decryptSubStreamInterface interface {
	getDecrypteSubStream() *DecryptStream
}

// getPermuteSubStream returns the sub-stream, used to return an embedded struct
// off an interface.
func (ds *DecryptStream) getDecrypteSubStream() *DecryptStream {
	return ds
}

func (ds *DecryptStream) Input(index uint32, slot *mixmessages.Slot) error {

	if index >= uint32(ds.EcrMsg.Len()) {
		return node.ErrOutsideOfBatch
	}

	if !ds.Grp.BytesInside(slot.MessagePayload, slot.AssociatedData) {
		return node.ErrOutsideOfGroup
	}

	// Check that the user id is formatted correctly
	if len(slot.SenderID) != id.UserLen {
		return globals.ERR_NONEXISTANT_USER
	}

	// Check that the salt is formatted correctly
	if len(slot.Salt) != 32 {
		return globals.ERR_SALTINCORRECTLENGTH
	}

	//copy the user id
	copy((*ds.Users[index])[:], slot.SenderID[:])

	//link to the salt
	ds.Salts[index] = slot.Salt

	ds.Grp.SetBytes(ds.EcrMsg.Get(index), slot.MessagePayload)
	ds.Grp.SetBytes(ds.EcrAD.Get(index), slot.AssociatedData)
	return nil
}

func (ds *DecryptStream) Output(index uint32) *mixmessages.Slot {
	return &mixmessages.Slot{
		SenderID:       (*ds.Users[index])[:],
		Salt:           ds.Salts[index],
		MessagePayload: ds.EcrMsg.Get(index).Bytes(),
		AssociatedData: ds.EcrAD.Get(index).Bytes(),
	}
}

//module in realtime Decrypt implementing mul3
var DecryptMul3 = services.Module{
	// Multiplies in own Encrypted Keys and Partial Cypher Texts
	Adapt: func(streamInput services.Stream, cryptop cryptops.Cryptop, chunk services.Chunk) error {
		dssi, ok := streamInput.(decryptSubStreamInterface)
		mul3, ok2 := cryptop.(cryptops.Mul3Prototype)

		if !ok || !ok2 {
			return services.InvalidTypeAssert
		}

		ds := dssi.getDecrypteSubStream()

		for i := chunk.Begin(); i < chunk.End(); i++ {
			//Do mul3 ecrMessage=messageKey*R*ecrMessage%p
			mul3(ds.Grp, ds.KeysMsg.Get(i), ds.R.Get(i), ds.EcrMsg.Get(i))
			//Do mul3 ecrAD=ecrAD*U*ecrMessage%p
			mul3(ds.Grp, ds.KeysAD.Get(i), ds.U.Get(i), ds.EcrAD.Get(i))
		}
		return nil
	},
	Cryptop:    cryptops.Mul3,
	NumThreads: services.AUTO_NUMTHREADS,
	InputSize:  services.AUTO_INPUTSIZE,
	Name:       "DecryptMul3",
}

// InitDecryptGraph called to initialize the graph. Conforms to graphs.Initialize function type
func InitDecryptGraph(gc services.GraphGenerator) *services.Graph {
	g := gc.NewGraph("RealtimeDecrypt", &DecryptStream{})

	decryptKeygen := graphs.Keygen.DeepCopy()
	decryptMul3 := DecryptMul3.DeepCopy()

	g.First(decryptKeygen)
	g.Connect(decryptKeygen, decryptMul3)
	g.Last(decryptMul3)

	return g
}
