////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package globals

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/server/services"
	"strconv"
	"sync"
	"sync/atomic"
)

// Server-wide configured batch size
var BatchSize uint64

// LastNode contains precomputations held only by the last node
type LastNode struct {
	// Message Decryption key, AKA PiRST_Inv
	MessagePrecomputation []*cyclic.Int
	// Recipient ID Decryption Key, AKA PiUV_Inv
	RecipientPrecomputation []*cyclic.Int
	// Round Message Private Key
	RoundMessagePrivateKey []*cyclic.Int
	// Round Recipient Private Key
	RoundRecipientPrivateKey []*cyclic.Int

	// These are technically temp values, representing recipient info
	// Encrypted under homomorphic encryption that later get revealed
	RecipientCypherText              []*cyclic.Int
	EncryptedRecipientPrecomputation []*cyclic.Int
	EncryptedMessagePrecomputation   []*cyclic.Int

	// Temp value storing EncryptedMessages after RealtimePermute
	// in order to be passed into RealtimeEncrypt after RealtimeIdentify
	EncryptedMessage []*cyclic.Int
}

// Round contains the keys and permutations for a given message batch
type Round struct {
	R            []*cyclic.Int // First unpermuted internode message key
	S            []*cyclic.Int // Permuted internode message key
	T            []*cyclic.Int // Second unpermuted internode message key
	V            []*cyclic.Int // Unpermuted internode recipient key
	U            []*cyclic.Int // Permuted *cyclic.Internode receipient key
	R_INV        []*cyclic.Int // First Inverse unpermuted internode message key
	S_INV        []*cyclic.Int // Permuted Inverse internode message key
	T_INV        []*cyclic.Int // Second Inverse unpermuted internode message key
	V_INV        []*cyclic.Int // Unpermuted Inverse internode recipient key
	U_INV        []*cyclic.Int // Permuted Inverse *cyclic.Internode receipient key
	Permutations []uint64      // Permutation array, messages at index i become
	// messages at index Permutations[i]
	CypherPublicKey *cyclic.Int // Global Cypher Key
	Z               *cyclic.Int // This node's Cypher Key
	// Private keys for the above
	Y_R []*cyclic.Int
	Y_S []*cyclic.Int
	Y_T []*cyclic.Int
	Y_V []*cyclic.Int
	Y_U []*cyclic.Int

	// Variables only carried by the last node
	LastNode

	// Size of batch
	BatchSize uint64

	// Phase fields
	phase     Phase
	phaseCond *sync.Cond

	// Array of Channels associated to each Phase of this Round
	channels [NUM_PHASES]chan<- *services.Slot

	// Array of status booleans to store the results of the MIC
	MIC_Verification []bool
}

// Grp is the cyclic group that all operations are done within
var Grp *cyclic.Group

// Global instance of RoundMap
var GlobalRoundMap RoundMap

// Atomic counter to generate round IDs
var globalRoundCounter uint64

func getAndIncrementRoundCounter() uint64 {
	defer atomic.AddUint64(&globalRoundCounter, uint64(1))
	return globalRoundCounter
}

// FIXME, maybe: This is used by last node to precalc the round id
func PeekNextRoundID() string {
	return strconv.FormatUint(globalRoundCounter, 36)
}

// TODO: have a better way to generate round IDs
func GetNextRoundID() string {
	// 36 is the base for formatting
	return strconv.FormatUint(getAndIncrementRoundCounter(), 36)
}

// Wrapper struct for a map of String -> Round structs
type RoundMap struct {
	// Mapping of session identifiers to round structures
	rounds map[string]*Round
	// Mutex for atomic get/add operations (Automatically initiated)
	mutex sync.Mutex
}

// Create and return a new RoundMap with initialized fields
func NewRoundMap() RoundMap {
	return RoundMap{rounds: make(map[string]*Round)}
}

// Atomic get *Round for a given roundId in rounds map
func (m *RoundMap) GetRound(roundId string) *Round {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return m.rounds[roundId]
}

// Atomic add *Round to rounds map with given roundId
func (m *RoundMap) AddRound(roundId string, newRound *Round) {
	m.mutex.Lock()
	m.rounds[roundId] = newRound
	m.mutex.Unlock()
}

// Atomic delete *Round to rounds map with given roundId
func (m *RoundMap) DeleteRound(roundId string) {
	m.mutex.Lock()
	delete(m.rounds, roundId)
	m.mutex.Unlock()
}

// Get chan for a given chanId in channels array (Not thread-safe!)
func (round *Round) GetChannel(chanId Phase) chan<- *services.Slot {
	return round.channels[chanId]
}

// Add chan to channels array with given chanId (Not thread-safe!)
func (round *Round) AddChannel(chanId Phase, newChan chan<- *services.Slot) {
	round.channels[chanId] = newChan
}

// Returns when the provided round reaches the specified phase
// Returns immediately if the phase has already past or it is in
// an error state.
func (round *Round) WaitUntilPhase(phase Phase) {
	round.phaseCond.L.Lock() // This must be held when calling wait
	for round.phase < phase {
		jww.DEBUG.Printf("Current Phase State: %s", round.phase.String())
		round.phaseCond.Wait()
	}
	round.phaseCond.L.Unlock()
}

// NewRound constructs an empty round for a given batch size, with all
// numbers being initialized to 0.
func NewRound(batchSize uint64) *Round {
	return newRound(batchSize, OFF)
}

//Creates a new Round at any phase
func NewRoundWithPhase(batchSize uint64, p Phase) *Round {
	return newRound(batchSize, p)
}

// Returns a copy of the current phase
func (round *Round) GetPhase() Phase {
	round.phaseCond.L.Lock()
	p := round.phase
	round.phaseCond.L.Unlock()
	return p
}

// Sets the phase, and signals the phaseCond that the phase state has changed
// Note that phases can only advance state, and can sometimes skip state when
// the node is not the last node.
func (round *Round) SetPhase(p Phase) {
	round.phaseCond.L.Lock()
	if p < round.phase {
		jww.FATAL.Panicf("Cannot decrement Phases!")
	}
	round.phase = p
	round.phaseCond.L.Unlock()
	round.phaseCond.Signal()
}

// Unexported underlying function to initialize a new round
func newRound(batchSize uint64, p Phase) *Round {
	NR := Round{
		R: make([]*cyclic.Int, batchSize),
		S: make([]*cyclic.Int, batchSize),
		T: make([]*cyclic.Int, batchSize),
		V: make([]*cyclic.Int, batchSize),
		U: make([]*cyclic.Int, batchSize),

		R_INV: make([]*cyclic.Int, batchSize),
		S_INV: make([]*cyclic.Int, batchSize),
		T_INV: make([]*cyclic.Int, batchSize),
		V_INV: make([]*cyclic.Int, batchSize),
		U_INV: make([]*cyclic.Int, batchSize),

		CypherPublicKey: cyclic.NewMaxInt(),
		Z:               cyclic.NewMaxInt(),

		Permutations: make([]uint64, batchSize),

		Y_R: make([]*cyclic.Int, batchSize),
		Y_S: make([]*cyclic.Int, batchSize),
		Y_T: make([]*cyclic.Int, batchSize),
		Y_V: make([]*cyclic.Int, batchSize),
		Y_U: make([]*cyclic.Int, batchSize),

		BatchSize: batchSize,

		MIC_Verification: make([]bool, batchSize)}

	NR.CypherPublicKey.SetBytes(cyclic.Max4kBitInt)
	NR.Z.SetBytes(cyclic.Max4kBitInt)

	for i := uint64(0); i < batchSize; i++ {
		NR.R[i] = cyclic.NewMaxInt()
		NR.S[i] = cyclic.NewMaxInt()
		NR.T[i] = cyclic.NewMaxInt()
		NR.V[i] = cyclic.NewMaxInt()
		NR.U[i] = cyclic.NewMaxInt()

		NR.R_INV[i] = cyclic.NewMaxInt()
		NR.S_INV[i] = cyclic.NewMaxInt()
		NR.T_INV[i] = cyclic.NewMaxInt()
		NR.V_INV[i] = cyclic.NewMaxInt()
		NR.U_INV[i] = cyclic.NewMaxInt()

		NR.Y_R[i] = cyclic.NewMaxInt()
		NR.Y_S[i] = cyclic.NewMaxInt()
		NR.Y_T[i] = cyclic.NewMaxInt()
		NR.Y_V[i] = cyclic.NewMaxInt()
		NR.Y_U[i] = cyclic.NewMaxInt()

		NR.Permutations[i] = i

		NR.LastNode.MessagePrecomputation = nil
		NR.LastNode.RecipientPrecomputation = nil

		NR.MIC_Verification[i] = true
	}
	NR.phase = p
	NR.phaseCond = &sync.Cond{L: &sync.Mutex{}}

	return &NR
}

func ResetRound(NR *Round, batchSize uint64) {
	NR.CypherPublicKey.SetBytes(cyclic.Max4kBitInt)
	NR.Z.SetBytes(cyclic.Max4kBitInt)

	for i := uint64(0); i < batchSize; i++ {
		NR.R[i].SetBytes(cyclic.Max4kBitInt)
		NR.S[i].SetBytes(cyclic.Max4kBitInt)
		NR.T[i].SetBytes(cyclic.Max4kBitInt)
		NR.V[i].SetBytes(cyclic.Max4kBitInt)
		NR.U[i].SetBytes(cyclic.Max4kBitInt)

		NR.R_INV[i].SetBytes(cyclic.Max4kBitInt)
		NR.S_INV[i].SetBytes(cyclic.Max4kBitInt)
		NR.T_INV[i].SetBytes(cyclic.Max4kBitInt)
		NR.V_INV[i].SetBytes(cyclic.Max4kBitInt)
		NR.U_INV[i].SetBytes(cyclic.Max4kBitInt)

		NR.Y_R[i].SetBytes(cyclic.Max4kBitInt)
		NR.Y_S[i].SetBytes(cyclic.Max4kBitInt)
		NR.Y_T[i].SetBytes(cyclic.Max4kBitInt)
		NR.Y_V[i].SetBytes(cyclic.Max4kBitInt)
		NR.Y_U[i].SetBytes(cyclic.Max4kBitInt)

		NR.Permutations[i] = i

		NR.LastNode.MessagePrecomputation = nil
		NR.LastNode.RecipientPrecomputation = nil

		NR.MIC_Verification[i] = true
	}
	NR.phase = OFF
	NR.phaseCond = &sync.Cond{L: &sync.Mutex{}}

	for i := Phase(0); i < NUM_PHASES; i++ {
		NR.channels[i] = nil
	}
}

func InitLastNode(round *Round) {
	round.LastNode.MessagePrecomputation = make([]*cyclic.Int, round.BatchSize)
	round.LastNode.RecipientPrecomputation = make([]*cyclic.Int, round.BatchSize)
	round.LastNode.RoundMessagePrivateKey = make([]*cyclic.Int, round.BatchSize)
	round.LastNode.RoundRecipientPrivateKey = make([]*cyclic.Int, round.BatchSize)
	round.LastNode.RecipientCypherText = make([]*cyclic.Int, round.BatchSize)
	round.LastNode.EncryptedRecipientPrecomputation = make([]*cyclic.Int,
		round.BatchSize)
	round.LastNode.EncryptedMessagePrecomputation = make([]*cyclic.Int,
		round.BatchSize)
	round.LastNode.EncryptedMessage = make([]*cyclic.Int, round.BatchSize)

	for i := uint64(0); i < round.BatchSize; i++ {
		round.LastNode.MessagePrecomputation[i] = cyclic.NewMaxInt()
		round.LastNode.RecipientPrecomputation[i] = cyclic.NewMaxInt()
		round.LastNode.RoundMessagePrivateKey[i] = cyclic.NewMaxInt()
		round.LastNode.RoundRecipientPrivateKey[i] = cyclic.NewMaxInt()
		round.LastNode.RecipientCypherText[i] = cyclic.NewMaxInt()
		round.LastNode.EncryptedRecipientPrecomputation[i] = cyclic.NewMaxInt()
		round.LastNode.EncryptedMessagePrecomputation[i] = cyclic.NewMaxInt()
		round.LastNode.EncryptedMessage[i] = cyclic.NewMaxInt()
		round.MIC_Verification[i] = false
	}
}

func ResetLastNode(round *Round) {

	for i := uint64(0); i < round.BatchSize; i++ {
		round.LastNode.MessagePrecomputation[i].SetBytes(cyclic.Max4kBitInt)
		round.LastNode.RecipientPrecomputation[i].SetBytes(cyclic.Max4kBitInt)
		round.LastNode.RoundMessagePrivateKey[i].SetBytes(cyclic.Max4kBitInt)
		round.LastNode.RoundRecipientPrivateKey[i].SetBytes(cyclic.Max4kBitInt)
		round.LastNode.RecipientCypherText[i].SetBytes(cyclic.Max4kBitInt)
		round.LastNode.EncryptedRecipientPrecomputation[i].SetBytes(cyclic.Max4kBitInt)
		round.LastNode.EncryptedMessagePrecomputation[i].SetBytes(cyclic.Max4kBitInt)
		round.LastNode.EncryptedMessage[i].SetBytes(cyclic.Max4kBitInt)
		round.MIC_Verification[i] = false
	}
}
