package cmix

import (
	"gitlab.com/elixxir/crypto/cyclic"
	"sync"
	"time"
	"gitlab.com/elixxir/server/services"
)

type RoundID uint64

type Round struct {
	id RoundID

	R     cyclic.IntBuffer // First unpermuted internode message key
	S     cyclic.IntBuffer // Permuted internode message key
	T     cyclic.IntBuffer // Second unpermuted internode message key
	V     cyclic.IntBuffer // Unpermuted internode associated data key
	U     cyclic.IntBuffer // Permuted *cyclic.Internode recipient key
	R_INV cyclic.IntBuffer // First Inverse unpermuted internode message key
	S_INV cyclic.IntBuffer // Permuted Inverse internode message key
	T_INV cyclic.IntBuffer // Second Inverse unpermuted internode message key
	V_INV cyclic.IntBuffer // Unpermuted Inverse internode associated data key
	U_INV cyclic.IntBuffer // Permuted Inverse *cyclic.Internode recipient key

	CypherPublicKey *cyclic.Int // Global Cypher Key
	Z               *cyclic.Int // This cmix's Cypher Key

	// Private keys for the above
	Y_R []*cyclic.Int
	Y_S []*cyclic.Int
	Y_T []*cyclic.Int
	Y_V []*cyclic.Int
	Y_U []*cyclic.Int

	// Size of batch
	batchSize         uint32
	expandedBatchSize uint32

	// Map of graphs which implement phases
	phases PhaseMap
}

// Function to initialize a new round
func NewRound(g *cyclic.Group, batchsize, expandedBatchSize uint32) *Round {
	NR := Round{
		R: g.NewIntBuffer(expandedBatchSize),
		S: make([]*cyclic.Int, batchSize),
		T: make([]*cyclic.Int, batchSize),
		V: make([]*cyclic.Int, batchSize),
		U: make([]*cyclic.Int, batchSize),

		R_INV: make([]*cyclic.Int, batchSize),
		S_INV: make([]*cyclic.Int, batchSize),
		T_INV: make([]*cyclic.Int, batchSize),
		V_INV: make([]*cyclic.Int, batchSize),
		U_INV: make([]*cyclic.Int, batchSize),

		CypherPublicKey: g.NewMaxInt(),
		Z:               g.NewMaxInt(),

		Permutations: make([]uint64, batchSize),

		Y_R: make([]*cyclic.Int, batchSize),
		Y_S: make([]*cyclic.Int, batchSize),
		Y_T: make([]*cyclic.Int, batchSize),
		Y_V: make([]*cyclic.Int, batchSize),
		Y_U: make([]*cyclic.Int, batchSize),

		BatchSize: batchSize,
		ExpSize:   uint32(256),

		MIC_Verification: make([]bool, batchSize),
	}

	g.Set(NR.CypherPublicKey, g.NewMaxInt())
	g.Set(NR.Z, g.NewMaxInt())

	for i := uint64(0); i < batchSize; i++ {
		NR.R[i] = g.NewMaxInt()
		NR.S[i] = g.NewMaxInt()
		NR.T[i] = g.NewMaxInt()
		NR.V[i] = g.NewMaxInt()
		NR.U[i] = g.NewMaxInt()

		NR.R_INV[i] = g.NewMaxInt()
		NR.S_INV[i] = g.NewMaxInt()
		NR.T_INV[i] = g.NewMaxInt()
		NR.V_INV[i] = g.NewMaxInt()
		NR.U_INV[i] = g.NewMaxInt()

		NR.Y_R[i] = g.NewMaxInt()
		NR.Y_S[i] = g.NewMaxInt()
		NR.Y_T[i] = g.NewMaxInt()
		NR.Y_V[i] = g.NewMaxInt()
		NR.Y_U[i] = g.NewMaxInt()

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
