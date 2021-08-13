///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package round

// buffer.go contains the round.Buffer object. It also contains it's methods
// and constructors

import (
	"github.com/pkg/errors"
	pb "git.xx.network/elixxir/comms/mixmessages"
	"git.xx.network/elixxir/crypto/cyclic"
	"git.xx.network/xx_network/primitives/id"
	"sync"
	"sync/atomic"
)

type Buffer struct {
	batchSize         uint32
	expandedBatchSize uint32

	// BatchWide Keys
	CypherPublicKey *cyclic.Int // Global Cypher Key
	Z               *cyclic.Int // This node's private Cypher Key

	//Realtime Keys
	R *cyclic.IntBuffer // First unpermuted internode payloadA key
	S *cyclic.IntBuffer // Permuted internode payloadA key
	U *cyclic.IntBuffer // Permuted internode payloadB key
	V *cyclic.IntBuffer // Unpermuted internode payloadB key

	// Private keys for the above
	Y_R *cyclic.IntBuffer
	Y_S *cyclic.IntBuffer
	Y_T *cyclic.IntBuffer
	Y_V *cyclic.IntBuffer
	Y_U *cyclic.IntBuffer

	// Pre-populated permutations
	Permutations []uint32

	// Results of Precomputation
	PayloadAPrecomputation *cyclic.IntBuffer
	PayloadBPrecomputation *cyclic.IntBuffer

	// Stores the result of the precomputation permuted phase for the last node
	// To reuse in the Identify phase because the Reveal phase does not use the data
	PermutedPayloadAKeys []*cyclic.Int
	PermutedPayloadBKeys []*cyclic.Int

	// Multiparty DH Keys
	FinalKeys          []*cyclic.Int
	FinalShareMessages map[*id.ID]*pb.SharePiece
	SharesReceived     *uint32
	SharePhaseMux      sync.RWMutex
}

// Function to initialize a new round
func NewBuffer(g *cyclic.Group, batchSize, expandedBatchSize uint32) *Buffer {

	permutations := make([]uint32, expandedBatchSize)
	for i := uint32(0); i < expandedBatchSize; i++ {
		permutations[i] = i
	}
	newSharedMessageMap := make(map[*id.ID]*pb.SharePiece)
	sharedReceived := uint32(0)
	return &Buffer{
		R: g.NewIntBuffer(expandedBatchSize, g.NewInt(1)),
		S: g.NewIntBuffer(expandedBatchSize, g.NewInt(1)),
		V: g.NewIntBuffer(expandedBatchSize, g.NewInt(1)),
		U: g.NewIntBuffer(expandedBatchSize, g.NewInt(1)),

		CypherPublicKey: g.NewMaxInt(),
		Z:               g.NewMaxInt(),

		Y_R: g.NewIntBuffer(expandedBatchSize, g.NewInt(1)),
		Y_S: g.NewIntBuffer(expandedBatchSize, g.NewInt(1)),
		Y_T: g.NewIntBuffer(expandedBatchSize, g.NewInt(1)),
		Y_V: g.NewIntBuffer(expandedBatchSize, g.NewInt(1)),
		Y_U: g.NewIntBuffer(expandedBatchSize, g.NewInt(1)),

		Permutations: permutations,

		batchSize:         batchSize,
		expandedBatchSize: expandedBatchSize,

		PayloadAPrecomputation: g.NewIntBuffer(expandedBatchSize, g.NewInt(1)),
		PayloadBPrecomputation: g.NewIntBuffer(expandedBatchSize, g.NewInt(1)),

		FinalKeys:          make([]*cyclic.Int, 0),
		SharesReceived:     &sharedReceived,
		FinalShareMessages: newSharedMessageMap,
		SharePhaseMux:      sync.RWMutex{},
	}
}

func (r *Buffer) InitLastNode() {
	r.PermutedPayloadAKeys = make([]*cyclic.Int, r.expandedBatchSize)
	r.PermutedPayloadBKeys = make([]*cyclic.Int, r.expandedBatchSize)
}

func (r *Buffer) GetBatchSize() uint32 {
	return r.batchSize
}

func (r *Buffer) GetExpandedBatchSize() uint32 {
	return r.expandedBatchSize
}

// Erase clears all data contained in the buffer. All elements are set to zero
// and all arrays are set to nil. All underlying released data will be removed
// by the garbage collector.
func (r *Buffer) Erase() {
	r.batchSize = 0
	r.expandedBatchSize = 0

	r.CypherPublicKey.Erase()
	r.Z.Erase()

	r.R.Erase()
	r.S.Erase()
	r.U.Erase()
	r.V.Erase()

	r.Y_R.Erase()
	r.Y_S.Erase()
	r.Y_T.Erase()
	r.Y_V.Erase()
	r.Y_U.Erase()

	r.Permutations = nil

	r.PayloadAPrecomputation.Erase()
	r.PayloadBPrecomputation.Erase()

	r.PermutedPayloadAKeys = nil
	r.PermutedPayloadBKeys = nil

	atomic.SwapUint32(r.SharesReceived, 0)
	r.FinalKeys = nil
	for key := range r.FinalShareMessages {
		delete(r.FinalShareMessages, key)
	}

}

// AddFinalShareMessage adds to the message tracker final share piece from the sender
// If we have a message in there already, we error
func (r *Buffer) AddFinalShareMessage(piece *pb.SharePiece, origin *id.ID) error {
	r.SharePhaseMux.Lock()
	defer r.SharePhaseMux.Unlock()
	// Check that we have not already received a final piece message from the sender
	if r.FinalShareMessages[origin] != nil {
		return errors.Errorf("Received final key from [%s] multiple times",
			origin)
	}
	r.FinalShareMessages[origin] = piece
	return nil
}

// GetPieceMessagesByNode gets final share piece message received by the
// specified nodeID
func (r *Buffer) GetPieceMessagesByNode(origin *id.ID) *pb.SharePiece {
	r.SharePhaseMux.RLock()
	messages := r.FinalShareMessages[origin]
	r.SharePhaseMux.RUnlock()
	return messages
}

// UpdateFinalKeys adds a new key to the list of final keys
func (r *Buffer) UpdateFinalKeys(piece *cyclic.Int) []*cyclic.Int {
	r.SharePhaseMux.Lock()
	defer r.SharePhaseMux.Unlock()
	r.FinalKeys = append(r.FinalKeys, piece)
	return r.FinalKeys
}

// Increments the number of shares received
// as part of phaseShare
func (r *Buffer) IncrementShares() uint32 {
	return atomic.AddUint32(r.SharesReceived, 1)
}
