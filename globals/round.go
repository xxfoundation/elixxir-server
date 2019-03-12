////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package globals

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/services"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

// Server-wide configured batch size
var BatchSize uint64

// Server-wide gateway
var GatewayAddress string = ""

var RoundRecycle chan *Round

// LastNode contains precomputations held only by the last node
type LastNode struct {
	// Message Decryption key, AKA PiRST_Inv
	MessagePrecomputation []*cyclic.Int
	// AssociatedData Decryption Key, AKA PiUV_Inv
	AssociatedDataPrecomputation []*cyclic.Int
	// Round Message Private Key
	RoundMessagePrivateKey []*cyclic.Int
	// Round AssociatedData Private Key
	RoundAssociatedDataPrivateKey []*cyclic.Int

	// These are technically temp values, representing associated data
	// Encrypted under homomorphic encryption that later get revealed
	AssociatedDataCypherText              []*cyclic.Int
	EncryptedAssociatedDataPrecomputation []*cyclic.Int
	EncryptedMessagePrecomputation        []*cyclic.Int

	// Temp value storing EncryptedMessages after RealtimePermute
	// in order to be passed into RealtimeEncrypt after RealtimeIdentify
	EncryptedMessage []*cyclic.Int
}

// Round contains the keys and permutations for a given message batch
type Round struct {
	R            []*cyclic.Int // First unpermuted internode message key
	S            []*cyclic.Int // Permuted internode message key
	T            []*cyclic.Int // Second unpermuted internode message key
	V            []*cyclic.Int // Unpermuted internode associated data key
	U            []*cyclic.Int // Permuted *cyclic.Internode receipient key
	R_INV        []*cyclic.Int // First Inverse unpermuted internode message key
	S_INV        []*cyclic.Int // Permuted Inverse internode message key
	T_INV        []*cyclic.Int // Second Inverse unpermuted internode message key
	V_INV        []*cyclic.Int // Unpermuted Inverse internode associated data key
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

	// Size of exponents in bits
	ExpSize uint32

	// Phase fields
	phase     Phase
	phaseCond *sync.Cond

	// Array of Channels associated to each Phase of this Round
	channels [NUM_PHASES]chan<- *services.Slot

	// Array of status booleans to store the results of the MIC
	MIC_Verification []bool

	//Stores the start times for computations so they can be evaluated
	CryptopStartTimes [NUM_PHASES]time.Time
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
	round := m.rounds[roundId]
	m.mutex.Unlock()
	return round
}

// Atomic SetPhase -- NOTE: If round id was removed from the map this does
// nothing.
func (m *RoundMap) SetPhase(roundId string, p Phase) *Round {
	m.mutex.Lock()
	round, ok := m.rounds[roundId]
	if ok {
		round.SetPhase(p)
	}
	m.mutex.Unlock()
	return round
}

// Atomic add *Round to rounds map with given roundId
func (m *RoundMap) AddRound(roundId string, newRound *Round) {
	m.mutex.Lock()
	m.rounds[roundId] = newRound
	m.mutex.Unlock()
}

// Atomic delete *Round to rounds map with given roundId
func (m *RoundMap) DeleteRound(roundId string) {
	round := m.GetRound(roundId)
	m.mutex.Lock()
	delete(m.rounds, roundId)
	// FIXME: Disabling round recycling until we can do it right.
	round.SetPhase(REAL_COMPLETE)
	// ResetRound(round)
	// RoundRecycle <- round
	m.mutex.Unlock()
	jww.INFO.Printf("Round %v has been recycled", roundId)
}

// Get chan for a given chanId in channels array (Not thread-safe!)
func (round *Round) GetChannel(chanId Phase) chan<- *services.Slot {
	if round == nil {
		n := make(chan *services.Slot, BatchSize)
		go func(nullChan chan *services.Slot) {
			for elem := range nullChan {
				// ignore it.
				jww.WARN.Printf("Dropping value %v", elem)
			}
		}(n)
		return n
	}
	return round.channels[chanId]
}

// Add chan to channels array with given chanId (Not thread-safe!)
func (round *Round) AddChannel(chanId Phase, newChan chan<- *services.Slot) {
	if round == nil {
		return
	}
	round.channels[chanId] = newChan
}

// Returns when the provided round reaches the specified phase
// Returns immediately if the phase has already past or it is in
// an error state.
func (round *Round) WaitUntilPhase(phase Phase) {
	if round == nil {
		return
	}
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
	var round *Round
	select {
	case round = <-RoundRecycle:
	default:
		round = newRound(batchSize, OFF)
		if id.IsLastNode {
			InitLastNode(round)
		}
	}
	round = newRound(batchSize, OFF)
	if id.IsLastNode {
		InitLastNode(round)
	}

	return round
}

//Creates a new Round at any phase
func NewRoundWithPhase(batchSize uint64, p Phase) *Round {
	return newRound(batchSize, p)
}

// Returns a copy of the current phase
func (round *Round) GetPhase() Phase {
	if round == nil {
		return ERROR
	}
	round.phaseCond.L.Lock()
	p := round.phase
	round.phaseCond.L.Unlock()
	return p
}

// Sets the phase, and signals the phaseCond that the phase state has changed
// Note that phases can only advance state, and can sometimes skip state when
// the node is not the last node.
func (round *Round) SetPhase(p Phase) {
	if round == nil {
		return
	}
	jww.INFO.Printf("Setting phase to %v", p.String())
	round.phaseCond.L.Lock()
	// These calls must be deferred so that they're still called after the panic
	defer func() {
		round.phaseCond.L.Unlock()
		round.phaseCond.Broadcast()
	}()
	if p < round.phase && round.phase != ERROR {
		jww.FATAL.Panicf("Cannot decrement Phases!")
	}
	round.phase = p
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
		ExpSize:   uint32(256),

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
		NR.LastNode.AssociatedDataPrecomputation = nil

		NR.MIC_Verification[i] = true
	}
	NR.phase = p
	NR.phaseCond = &sync.Cond{L: &sync.Mutex{}}

	for i := 0; i < int(NUM_PHASES); i++ {
		NR.CryptopStartTimes[i] = time.Now()
	}

	return &NR
}

func ResetRound(NR *Round) {
	batchSize := NR.BatchSize
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

		NR.MIC_Verification[i] = true
	}
	NR.SetPhase(REAL_COMPLETE)
	NR.phaseCond.L.Lock()
	NR.phase = OFF
	NR.phaseCond.L.Unlock()
	NR.phaseCond.Broadcast()

	for i := Phase(0); i < NUM_PHASES; i++ {
		NR.channels[i] = nil
	}
}

func InitLastNode(round *Round) {
	round.LastNode.MessagePrecomputation = make([]*cyclic.Int, round.BatchSize)
	round.LastNode.AssociatedDataPrecomputation = make([]*cyclic.Int, round.BatchSize)
	round.LastNode.RoundMessagePrivateKey = make([]*cyclic.Int, round.BatchSize)
	round.LastNode.RoundAssociatedDataPrivateKey = make([]*cyclic.Int, round.BatchSize)
	round.LastNode.AssociatedDataCypherText = make([]*cyclic.Int, round.BatchSize)
	round.LastNode.EncryptedAssociatedDataPrecomputation = make([]*cyclic.Int,
		round.BatchSize)
	round.LastNode.EncryptedMessagePrecomputation = make([]*cyclic.Int,
		round.BatchSize)
	round.LastNode.EncryptedMessage = make([]*cyclic.Int, round.BatchSize)

	for i := uint64(0); i < round.BatchSize; i++ {
		round.LastNode.MessagePrecomputation[i] = cyclic.NewMaxInt()
		round.LastNode.AssociatedDataPrecomputation[i] = cyclic.NewMaxInt()
		round.LastNode.RoundMessagePrivateKey[i] = cyclic.NewMaxInt()
		round.LastNode.RoundAssociatedDataPrivateKey[i] = cyclic.NewMaxInt()
		round.LastNode.AssociatedDataCypherText[i] = cyclic.NewMaxInt()
		round.LastNode.EncryptedAssociatedDataPrecomputation[i] = cyclic.NewMaxInt()
		round.LastNode.EncryptedMessagePrecomputation[i] = cyclic.NewMaxInt()
		round.LastNode.EncryptedMessage[i] = cyclic.NewMaxInt()
		round.MIC_Verification[i] = false
	}
}
