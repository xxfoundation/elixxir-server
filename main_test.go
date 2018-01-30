package main

package services

import (
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/server/node"
	"gitlab.com/privategrity/server/cryptops/precomputation"
	"gitlab.com/privategrity/server/cryptops/realtime"
	"testing"
	"time"
)

// Perform an end to end test of the precomputation with batchsize 1,
// then use it to send the message through a 1-node system to smoke test
// the cryptographic operations.
func TestEndToEndCryptops(t *testing.T) {
	batchSize := 1
	rng := cyclic.NewRandom(cyclic.NewInt(0), cyclic.NewInt(1000))
	grp := cyclic.NewGroup(cyclic.NewInt(11), cyclic.NewInt(5), cyclic.NewInt(12),
		rng)
	round := node.NewRound(batchSize)
	round.CypherPublicKey = cyclic.NewInt(3)

	// ----- PRECOMPUTATION ----- //

	// GENERATION PHASE
	// This phase requires us to use pre-cooked crypto values. We run
	// the step here then overwrite the values that were stored in the
	// round structure so we still get the same results.
	

	// SHARE PHASE

	// ENCRYPTION PHASE

	// DECRYPT PHASE

	// PERMUTE PHASE

	// ENCRYPT PHASE

	// REVEAL PHASE

	// STRIP PHASE

	// ----- REALTIME ----- //

	// DECRYPT PHASE

	// PERMUTE PHASE

	// IDENTIFY PHASE

	// ENCRYPT PHASE

	// PEEL PHASE
}
