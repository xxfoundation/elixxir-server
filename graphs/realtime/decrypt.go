////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package realtime

// Realtime Decrypt

import (
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/nike"
	"gitlab.com/elixxir/crypto/nike/ecdh"
	"gitlab.com/elixxir/gpumathsgo"
	"gitlab.com/elixxir/gpumathsgo/cryptops"
	"gitlab.com/elixxir/server/graphs"
	"gitlab.com/elixxir/server/internal/round"
	"gitlab.com/elixxir/server/services"
	"gitlab.com/elixxir/server/storage"
	"gitlab.com/xx_network/primitives/id"
)

const (
	RoundBuff = 0
)

// KeygenDecryptStream holds data containing keys and inputs used by decrypt
type KeygenDecryptStream struct {
	Grp        *cyclic.Group
	StreamPool *gpumaths.StreamPool
	PrivKeyPem []byte

	// Link to round object
	R *cyclic.IntBuffer
	U *cyclic.IntBuffer

	// Unique to stream
	EcrPayloadA *cyclic.IntBuffer
	EcrPayloadB *cyclic.IntBuffer

	// Components for key generation
	Users        []*id.ID
	Salts        [][]byte
	KeysPayloadA *cyclic.IntBuffer
	KeysPayloadB *cyclic.IntBuffer
	KMACS        [][][]byte

	graphs.KeygenSubStream
}

// GetName returns stream name
func (s *KeygenDecryptStream) GetName() string {
	return "RealtimeDecryptStream"
}

// Link creates stream internal buffers and binds stream to local state objects in round
func (s *KeygenDecryptStream) Link(grp *cyclic.Group, batchSize uint32, source ...interface{}) {
	roundBuf := source[RoundBuff].(*round.Buffer)
	streamPool := source[2].(*gpumaths.StreamPool)
	// todo: have nodeSecret be hardcoded at this index once
	//  testing of databaseless client registration has been done
	//  This involves removing search for nodeSecret while
	//  iterating over source below
	users := make([]*id.ID, batchSize)
	var clientReporter *round.ClientReport
	var roundID id.Round
	var nodeSecret *storage.NodeSecretManager
	var precanStore *storage.PrecanStore
	// Find the client error reporter and the roundID (if it exists)
	var ok bool
	for _, face := range source {
		if _, ok = face.(*round.ClientReport); ok {
			clientReporter = face.(*round.ClientReport)
		}

		if _, ok = face.(id.Round); ok {
			roundID = face.(id.Round)
		}

		if _, ok = face.(*storage.NodeSecretManager); ok {
			nodeSecret = face.(*storage.NodeSecretManager)
		}

		if _, ok = face.(*storage.PrecanStore); ok {
			precanStore = face.(*storage.PrecanStore)
		}
	}

	for i := uint32(0); i < batchSize; i++ {
		users[i] = &id.ID{}
	}

	s.LinkKeygenDecryptStream(grp, batchSize, roundBuf,
		streamPool, grp.NewIntBuffer(batchSize, grp.NewInt(1)),
		grp.NewIntBuffer(batchSize, grp.NewInt(1)),
		grp.NewIntBuffer(batchSize, grp.NewInt(1)),
		grp.NewIntBuffer(batchSize, grp.NewInt(1)),
		users, make([][]byte, batchSize), make([][][]byte, batchSize),
		make([][]bool, batchSize), make([]nike.PublicKey, batchSize), clientReporter, roundID,
		nodeSecret, precanStore)
}

// LinkKeygenDecryptStream creates stream internal buffers and binds stream to local state objects in round
func (s *KeygenDecryptStream) LinkKeygenDecryptStream(grp *cyclic.Group,
	batchSize uint32, round *round.Buffer, pool *gpumaths.StreamPool,
	ecrPayloadA, ecrPayloadB, keysPayloadA, keysPayloadB *cyclic.IntBuffer,
	users []*id.ID, salts [][]byte, kmacs [][][]byte, ephKeys [][]bool, clientEphemeralEd []nike.PublicKey,
	clientReporter *round.ClientReport, roundId id.Round,
	nodeSecrets *storage.NodeSecretManager,
	precanStore *storage.PrecanStore) {

	s.Grp = grp
	s.StreamPool = pool

	s.R = round.R.GetSubBuffer(0, batchSize)
	s.U = round.U.GetSubBuffer(0, batchSize)

	s.EcrPayloadA = ecrPayloadA
	s.EcrPayloadB = ecrPayloadB
	s.KeysPayloadA = keysPayloadA
	s.KeysPayloadB = keysPayloadB
	s.Users = users
	s.Salts = salts
	s.KMACS = kmacs
	s.NodeSecrets = nodeSecrets
	s.EphemeralKeys = ephKeys
	s.ClientEphemeralEd = clientEphemeralEd

	s.KeygenSubStream.LinkStream(s.Grp, s.Salts, s.KMACS, s.EphemeralKeys, s.ClientEphemeralEd,
		s.Users, s.KeysPayloadA, s.KeysPayloadB, clientReporter, roundId,
		batchSize, nodeSecrets, precanStore)
}

// KeygenDecryptSubStreamInterface creates an interface for KeygenDecryptStream to conform to
type KeygenDecryptSubStreamInterface interface {
	GetKeygenDecryptSubStream() *KeygenDecryptStream
}

// GetKeygenDecryptSubStream returns the sub-stream, used to return an embedded struct off an interface.
func (s *KeygenDecryptStream) GetKeygenDecryptSubStream() *KeygenDecryptStream {
	return s
}

// Input initializes stream inputs from slot received from IO
func (s *KeygenDecryptStream) Input(index uint32, slot *mixmessages.Slot) error {

	if index >= uint32(s.EcrPayloadA.Len()) {
		return services.ErrOutsideOfBatch
	}

	if !s.Grp.BytesInside(slot.PayloadA, slot.PayloadB) {
		return services.ErrOutsideOfGroup
	}

	// Check that the user id is formatted correctly
	if len(slot.SenderID) != id.ArrIDLen {
		return services.ErrUserIDTooShort
	}

	// Check that the salt is formatted correctly
	if len(slot.Salt) != 32 {
		return services.ErrSaltIncorrectLength
	}

	//copy the user id
	copy((*s.Users[index])[:], slot.SenderID[:])

	//link to the salt
	s.Salts[index] = slot.Salt

	//link to the KMACS
	s.KMACS[index] = slot.KMACs

	s.Grp.SetBytes(s.EcrPayloadA.Get(index), slot.PayloadA)
	s.Grp.SetBytes(s.EcrPayloadB.Get(index), slot.PayloadB)

	// Link to client ephemeral ED pubkey if relevant
	if slot.Ed25519 != nil {
		var err error
		s.ClientEphemeralEd[index], err = ecdh.ECDHNIKE.UnmarshalBinaryPublicKey(slot.Ed25519)
		if err != nil {
			return err
		}
	}
	// Link to ephemeralKeys.  If ephemeralkeys is not set, add an array of all false
	if slot.EphemeralKeys == nil || len(slot.EphemeralKeys) == 0 {
		slot.EphemeralKeys = make([]bool, len(slot.KMACs))
	}
	s.EphemeralKeys[index] = slot.EphemeralKeys

	return nil
}

// Output a cmix slot message for IO
func (s *KeygenDecryptStream) Output(index uint32) *mixmessages.Slot {
	var edBytes []byte
	if s.ClientEphemeralEd != nil && s.ClientEphemeralEd[index] != nil {
		edBytes = s.ClientEphemeralEd[index].Bytes()
	}
	return &mixmessages.Slot{
		Index:         index,
		SenderID:      (*s.Users[index])[:],
		Salt:          s.Salts[index],
		PayloadA:      s.EcrPayloadA.Get(index).Bytes(),
		PayloadB:      s.EcrPayloadB.Get(index).Bytes(),
		KMACs:         s.KMACS[index],
		EphemeralKeys: s.EphemeralKeys[index],
		Ed25519:       edBytes,
	}
}

// DecryptMul3 is the CPU module in realtime Decrypt implementing mul3
var DecryptMul3 = services.Module{
	// Multiplies in own Encrypted Keys and Partial Cypher Texts
	Adapt: func(streamInput services.Stream, cryptop cryptops.Cryptop, chunk services.Chunk) error {
		dssi, ok := streamInput.(KeygenDecryptSubStreamInterface)
		mul3, ok2 := cryptop.(cryptops.Mul3Prototype)

		if !ok || !ok2 {
			return services.InvalidTypeAssert
		}

		ds := dssi.GetKeygenDecryptSubStream()

		for i := chunk.Begin(); i < chunk.End(); i++ {
			//Do mul3 ecrPayloadA=payloadAKey*R*ecrPayloadA%p
			mul3(ds.Grp, ds.KeysPayloadA.Get(i), ds.R.Get(i), ds.EcrPayloadA.Get(i))
			//Do mul3 ecrPayloadB=payloadBKey*U*ecrPayloadB%p
			mul3(ds.Grp, ds.KeysPayloadB.Get(i), ds.U.Get(i), ds.EcrPayloadB.Get(i))
		}
		return nil
	},
	Cryptop:    cryptops.Mul3,
	NumThreads: services.AutoNumThreads,
	InputSize:  services.AutoInputSize,
	Name:       "DecryptMul3",
}

// DecryptMul3Chunk is the GPU module in realtime Decrypt implementing mul3
var DecryptMul3Chunk = services.Module{
	// Multiplies in own Encrypted Keys and Partial Cypher Texts
	Adapt: func(streamInput services.Stream, cryptop cryptops.Cryptop, chunk services.Chunk) error {
		dssi, ok := streamInput.(KeygenDecryptSubStreamInterface)
		mul3Chunk, ok2 := cryptop.(gpumaths.Mul3ChunkPrototype)

		if !ok || !ok2 {
			return services.InvalidTypeAssert
		}

		ds := dssi.GetKeygenDecryptSubStream()

		kpa := ds.KeysPayloadA.GetSubBuffer(chunk.Begin(), chunk.End())
		kpb := ds.KeysPayloadB.GetSubBuffer(chunk.Begin(), chunk.End())
		epa := ds.EcrPayloadA.GetSubBuffer(chunk.Begin(), chunk.End())
		epb := ds.EcrPayloadB.GetSubBuffer(chunk.Begin(), chunk.End())
		R := ds.R.GetSubBuffer(chunk.Begin(), chunk.End())
		U := ds.U.GetSubBuffer(chunk.Begin(), chunk.End())
		pool := ds.StreamPool
		grp := ds.Grp

		//Do mul3 ecrPayloadA=payloadAKey*R*ecrPayloadA%p
		err := mul3Chunk(pool, grp, kpa, R, epa, epa)
		if err != nil {
			return err
		}

		//Do mul3 ecrPayloadB=payloadBKey*U*ecrPayloadB%p
		err = mul3Chunk(pool, grp, kpb, U, epb, epb)
		if err != nil {
			return err
		}

		return nil
	},
	Cryptop:    gpumaths.Mul3Chunk,
	NumThreads: 2,
	InputSize:  32,
	Name:       "DecryptMul3Chunk",
}

// InitDecryptGraph is called to initialize the CPU Graph. Conforms to Graph.Initialize function type
func InitDecryptGraph(gc services.GraphGenerator) *services.Graph {
	if viper.GetBool("useGPU") {
		jww.FATAL.Panicf("Using realtime decrypt graph running on CPU instead of equivalent GPU graph")
	}
	g := gc.NewGraph("RealtimeDecrypt", &KeygenDecryptStream{})

	decryptKeygen := graphs.Keygen.DeepCopy()
	decryptMul3 := DecryptMul3.DeepCopy()

	g.First(decryptKeygen)
	g.Connect(decryptKeygen, decryptMul3)
	g.Last(decryptMul3)

	return g
}

// InitDecryptGPUGraph is called to initialize the GPU Graph. Conforms to Graph.Initialize function type
func InitDecryptGPUGraph(gc services.GraphGenerator) *services.Graph {
	if !viper.GetBool("useGPU") {
		jww.WARN.Printf("Using realtime decrypt graph running on GPU instead of equivalent CPU graph")
	}
	g := gc.NewGraph("RealtimeDecryptGPU", &KeygenDecryptStream{})

	decryptKeygen := graphs.Keygen.DeepCopy()
	decryptMul3Chunk := DecryptMul3Chunk.DeepCopy()

	g.First(decryptKeygen)
	g.Connect(decryptKeygen, decryptMul3Chunk)
	g.Last(decryptMul3Chunk)

	return g
}
