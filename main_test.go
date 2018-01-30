package main

import (
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/server/node"
	"gitlab.com/privategrity/server/services"
	"gitlab.com/privategrity/server/cryptops/precomputation"
	"fmt"
	"testing"
)

// Convert the round object into a string we can print
func RoundText(n *node.Round) string {
	outStr := fmt.Sprintf("\nCypherPublicKey: %s, Z: %s\n",
		n.CypherPublicKey.Text(10), n.Z.Text(10))
	outStr += fmt.Sprintf("Permutations: %v\n", n.Permutations)
	rfmt := "Round[%d]: \t R(%s, %s) S(%s, %s) T(%s, %s) \n" +
		"\t\t\t\t\t\t U(%s, %s) V(%s, %s) \n"
	for i := uint64(0); i < n.BatchSize; i++ {
		outStr += fmt.Sprintf(rfmt, i,
			n.R[i].Text(10), n.Y_R[i].Text(10),
			n.S[i].Text(10), n.Y_S[i].Text(10),
			n.T[i].Text(10), n.Y_T[i].Text(10),
			n.U[i].Text(10), n.Y_U[i].Text(10),
			n.V[i].Text(10), n.Y_V[i].Text(10))
	}
	return outStr
}

// Perform an end to end test of the precomputation with batchsize 1,
// then use it to send the message through a 1-node system to smoke test
// the cryptographic operations.
func TestEndToEndCryptops(t *testing.T) {
	batchSize := uint64(1)

	rng := cyclic.NewRandom(cyclic.NewInt(0), cyclic.NewInt(1000))
	grp := cyclic.NewGroup(cyclic.NewInt(11), cyclic.NewInt(5), cyclic.NewInt(12),
		rng)

	round := node.NewRound(batchSize)
	round.CypherPublicKey = cyclic.NewInt(3)

	// ----- PRECOMPUTATION ----- //

	// GENERATION PHASE
	Generation := services.DispatchCryptop(&grp, precomputation.Generation{},
		nil, nil, round)

	var inMessages []services.Slot
	for i := uint64(0); i < batchSize; i++ {
		//NOTE: This slot generation is vestigial and not really used..
		inMessages = append(inMessages, &precomputation.SlotGeneration{Slot: i})
	}

	// The following code kicks off the processing for generation, which we
	// dump to nowhere *because* we have to overwrite it.
	for i := uint64(0); i < batchSize; i++ {
		Generation.InChannel <- &(inMessages[i])
		_ = <-Generation.OutChannel
	}

	fmt.Printf("%v", RoundText(round))

	// TODO: This phase requires us to use pre-cooked crypto values. We run
	// the step here then overwrite the values that were stored in the
	// round structure so we still get the same results. We should perform
	// the override here.

	// SHARE PHASE
	var shareMsgs []services.Slot
	for i := uint64(0); i < batchSize; i++ {
		shareMsgs = append(shareMsgs,
			precomputation.SlotShare{Slot: i,
				PartialRoundPublicCypherKey: cyclic.NewInt(1)})
	}
	Share := services.DispatchCryptop(&grp, precomputation.Share{}, nil, nil,
		round)

	// DECRYPT PHASE
	Decrypt := services.DispatchCryptop(&grp, precomputation.Decrypt{},
		Share.OutChannel, nil, round)

	for i := uint64(0); i < batchSize; i++ {
		Share.InChannel <- &(shareMsgs[i])
		rtn := Decrypt.OutChannel
		t.Errorf("What? %v", rtn)
	}

	// PERMUTE PHASE

	// ENCRYPT PHASE

	// REVEAL PHASE

	// STRIP PHASE

	// KICK OFF PRECOMPUTATION
	for i := uint64(0); i < batchSize; i++ {
		Share.InChannel <- &(shareMsgs[i])
		rtn := Decrypt.OutChannel
		t.Errorf("What? %v", rtn)
	}

	// ----- REALTIME ----- //

	// DECRYPT PHASE

	// PERMUTE PHASE

	// IDENTIFY PHASE

	// ENCRYPT PHASE

	// PEEL PHASE

	// KICK OFF RT COMPUTATION
}
