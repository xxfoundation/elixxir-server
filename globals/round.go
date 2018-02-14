package globals

import (
	"errors"
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/server/services"
	"sync"
)

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
	phaseLock *sync.Mutex

	// Array of Channels associated to each Phase of this Round
	channels [NUM_PHASES]chan<- *services.Slot
}

// Grp is the cyclic group that all operations are done within
var Grp *cyclic.Group

// Global instance of RoundMap
var GlobalRoundMap RoundMap

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

// Get chan for a given chanId in channels array (Not thread-safe!)
func (round *Round) GetChannel(chanId Phase) chan<- *services.Slot {
	return round.channels[chanId]
}

// Add chan to channels array with given chanId (Not thread-safe!)
func (round *Round) AddChannel(chanId Phase, newChan chan<- *services.Slot) {
	round.channels[chanId] = newChan
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
	round.phaseLock.Lock()
	rp := round.phase
	round.phaseLock.Unlock()
	return rp
}

// Increments the phase if the phase can be incremented and was told to
// increment to the correct phase
func (round *Round) IncrementPhase(p Phase) error {
	round.phaseLock.Lock()

	if round.phase == DONE {
		round.phaseLock.Unlock()
		return errors.New("Cannot Increment Phase past DONE")
	}

	if round.phase == ERROR {
		round.phaseLock.Unlock()
		return errors.New("Cannot Increment a Phase in ERROR")
	}

	if (round.phase + 1) != p {
		round.phaseLock.Unlock()
		return errors.New("Invalid Phase Incrementation; Expected: " + (round.phase + 1).String() + ", Received: " + p.String())
	}

	round.phase++
	round.phaseLock.Unlock()

	return nil
}

// Puts the phase into an error state
func (round *Round) Error() {
	round.phaseLock.Lock()
	round.phase = ERROR
	round.phaseLock.Unlock()
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

		BatchSize: batchSize}

	NR.CypherPublicKey = cyclic.NewMaxInt()
	NR.Z = cyclic.NewMaxInt()

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
	}
	NR.phase = p
	NR.phaseLock = &sync.Mutex{}

	return &NR
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
	}
}
