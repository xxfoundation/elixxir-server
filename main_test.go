////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package main

import (
	jww "github.com/spf13/jwalterweatherman"
	//"gitlab.com/elixxir/crypto/cyclic"
	//"gitlab.com/elixxir/crypto/large"
	//"gitlab.com/elixxir/primitives/id"
	//"gitlab.com/elixxir/server/benchmark"
	//	"gitlab.com/elixxir/server/cryptops/precomputation"
	//	"gitlab.com/elixxir/server/cryptops/realtime"
	//"gitlab.com/elixxir/server/globals"
	//"gitlab.com/elixxir/server/services"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	// Set log level high for main testing to disable MIC errors, etc
	jww.SetStdoutThreshold(jww.LevelFatal)
	os.Exit(m.Run())
}

/*
// Perform an end to end test of the precomputation with batch size 1,
// then use it to send the message through a 2-node system to smoke test
// the cryptographic operations.
func TestEndToEndCryptopsWith2Nodes(t *testing.T) {

	// Init, we use a small prime to make it easier to run the numbers
	// when debugging
	batchSize := uint64(1)
	grp := cyclic.NewGroup(large.NewInt(107), large.NewInt(4), large.NewInt(5))
	Node1Round := globals.NewRound(batchSize, grp)
	Node2Round := globals.NewRound(batchSize, grp)
	Node1Round.CypherPublicKey = grp.NewInt(1)
	Node2Round.CypherPublicKey = grp.NewInt(1)

	// p=107 -> 7 bits, so exponents can be of 6 bits at most
	// Overwrite default value of rounds
	Node1Round.ExpSize = uint32(6)
	Node2Round.ExpSize = uint32(6)

	// Allocate the arrays for LastNode
	globals.InitLastNode(Node2Round, grp)

	// ----- PRECOMPUTATION ----- //
	N1Generation := services.DispatchCryptop(grp, precomputation.Generation{},
		nil, nil, Node1Round)
	N2Generation := services.DispatchCryptop(grp, precomputation.Generation{},
		nil, nil, Node2Round)
	// Since round.Z is generated on creation of the Generation precomp,
	// need to loop the generation here until a valid Z is produced
	maxInt := grp.NewMaxInt()
	for Node1Round.Z.Cmp(maxInt) == 0 || Node2Round.Z.Cmp(maxInt) == 0 {
		N1Generation = services.DispatchCryptop(grp, precomputation.Generation{},
			nil, nil, Node1Round)
		N2Generation = services.DispatchCryptop(grp, precomputation.Generation{},
			nil, nil, Node2Round)
	}

	N1Share := services.DispatchCryptop(grp, precomputation.Share{}, nil, nil,
		Node1Round)
	N2Share := services.DispatchCryptop(grp, precomputation.Share{},
		N1Share.OutChannel, nil, Node2Round)

	N1Decrypt := services.DispatchCryptop(grp, precomputation.Decrypt{},
		nil, nil, Node1Round)
	N2Decrypt := services.DispatchCryptop(grp, precomputation.Decrypt{},
		N1Decrypt.OutChannel, nil, Node2Round)

	N1Permute := services.DispatchCryptop(grp, precomputation.Permute{},
		N2Decrypt.OutChannel, nil, Node1Round)
	N2Permute := services.DispatchCryptop(grp, precomputation.Permute{},
		N1Permute.OutChannel, nil, Node2Round)

	N1Encrypt := services.DispatchCryptop(grp, precomputation.Encrypt{},
		nil, nil, Node1Round)
	N2Encrypt := services.DispatchCryptop(grp, precomputation.Encrypt{},
		N1Encrypt.OutChannel, nil, Node2Round)

	N1Reveal := services.DispatchCryptop(grp, precomputation.Reveal{},
		nil, nil, Node1Round)
	N2Reveal := services.DispatchCryptop(grp, precomputation.Reveal{},
		N1Reveal.OutChannel, nil, Node2Round)

	N2Strip := services.DispatchCryptop(grp, precomputation.Strip{},
		nil, nil, Node2Round)

	go benchmark.RevealStripTranslate(N2Reveal.OutChannel, N2Strip.InChannel)
	go benchmark.EncryptRevealTranslate(N2Encrypt.OutChannel, N1Reveal.InChannel,
		Node2Round, grp)
	go benchmark.PermuteEncryptTranslate(N2Permute.OutChannel, N1Encrypt.InChannel,
		Node2Round, grp)

	// Run Generate
	genMsg := services.Slot(&precomputation.SlotGeneration{Slot: 0})
	N1Generation.InChannel <- &genMsg
	_ = <-N1Generation.OutChannel
	N2Generation.InChannel <- &genMsg
	_ = <-N2Generation.OutChannel

	t.Logf("2 NODE GENERATION RESULTS: \n")
	t.Logf("%v", benchmark.RoundText(grp, Node1Round))
	t.Logf("%v", benchmark.RoundText(grp, Node2Round))

	// TODO: Pre-can the keys to use here if necessary.

	// Run Share -- Then save the result to both rounds
	// Note that the outchannel for N1Share is the input channel for N2share
	shareMsg := services.Slot(&precomputation.SlotShare{
		PartialRoundPublicCypherKey: grp.GetGCyclic()})
	N1Share.InChannel <- &shareMsg
	shareResultSlot := <-N2Share.OutChannel
	shareResult := (*shareResultSlot).(*precomputation.SlotShare)
	PublicCypherKey := grp.NewInt(1)
	grp.Set(PublicCypherKey, shareResult.PartialRoundPublicCypherKey)
	grp.Set(Node1Round.CypherPublicKey, PublicCypherKey)
	grp.Set(Node2Round.CypherPublicKey, PublicCypherKey)

	t.Logf("2 NODE SHARE RESULTS: \n")
	t.Logf("%v", benchmark.RoundText(grp, Node2Round))
	t.Logf("%v", benchmark.RoundText(grp, Node1Round))

	// Now finish precomputation
	decMsg := services.Slot(&precomputation.PrecomputationSlot{
		Slot:                         0,
		MessageCypher:                grp.NewInt(1),
		MessagePrecomputation:        grp.NewInt(1),
		AssociatedDataCypher:         grp.NewInt(1),
		AssociatedDataPrecomputation: grp.NewInt(1),
	})
	N1Decrypt.InChannel <- &decMsg
	rtn := <-N2Strip.OutChannel
	es := (*rtn).(*precomputation.PrecomputationSlot)

	Node2Round.LastNode.MessagePrecomputation[es.Slot] = es.MessagePrecomputation
	Node2Round.LastNode.AssociatedDataPrecomputation[es.Slot] =
		es.AssociatedDataPrecomputation
	t.Logf("2 NODE STRIP:\n  MessagePrecomputation: %s, "+
		"AssociatedDataPrecomputation: %s\n", es.MessagePrecomputation.Text(10),
		es.AssociatedDataPrecomputation.Text(10))

	// Check precomputation
	MP, RP := ComputePrecomputation(grp,
		[]*globals.Round{Node1Round, Node2Round})

	if MP.Cmp(es.MessagePrecomputation) != 0 {
		t.Errorf("Message Precomputation Incorrect! Expected: %s, Received: %s\n",
			MP.Text(10), es.MessagePrecomputation.Text(10))
	}
	if RP.Cmp(es.AssociatedDataPrecomputation) != 0 {
		t.Errorf("Recipient Precomputation Incorrect! Expected: %s, Received: %s\n",
			RP.Text(10), es.AssociatedDataPrecomputation.Text(10))
	}

	// ----- REALTIME ----- //
	IntermediateMsgs := make([]*cyclic.Int, 1)
	IntermediateMsgs[0] = grp.NewInt(1)

	N1RTDecrypt := services.DispatchCryptop(grp, realtime.Decrypt{},
		nil, nil, Node1Round)
	N2RTDecrypt := services.DispatchCryptop(grp, realtime.Decrypt{},
		nil, nil, Node2Round)

	N1RTPermute := services.DispatchCryptop(grp, realtime.Permute{},
		nil, nil, Node1Round)
	N2RTPermute := services.DispatchCryptop(grp, realtime.Permute{},
		N1RTPermute.OutChannel, nil, Node2Round)

	N2RTIdentify := services.DispatchCryptop(grp, realtime.Identify{},
		nil, nil, Node2Round)

	N1RTEncrypt := services.DispatchCryptop(grp, realtime.Encrypt{},
		nil, nil, Node1Round)
	N2RTEncrypt := services.DispatchCryptop(grp, realtime.Encrypt{},
		nil, nil, Node2Round)

	N2RTPeel := services.DispatchCryptop(grp, realtime.Peel{},
		nil, nil, Node2Round)

	go benchmark.RTEncryptRTEncryptTranslate(N1RTEncrypt.OutChannel,
		N2RTEncrypt.InChannel, grp)
	go benchmark.RTDecryptRTDecryptTranslate(N1RTDecrypt.OutChannel,
		N2RTDecrypt.InChannel, grp)
	go benchmark.RTDecryptRTPermuteTranslate(N2RTDecrypt.OutChannel,
		N1RTPermute.InChannel)
	go benchmark.RTPermuteRTIdentifyTranslate(N2RTPermute.OutChannel,
		N2RTIdentify.InChannel, IntermediateMsgs, grp)
	go benchmark.RTIdentifyRTEncryptTranslate(N2RTIdentify.OutChannel,
		N1RTEncrypt.InChannel, IntermediateMsgs, grp)
	go benchmark.RTEncryptRTPeelTranslate(N2RTEncrypt.OutChannel,
		N2RTPeel.InChannel)

	inputMsg := services.Slot(&realtime.Slot{
		Slot:           0,
		CurrentID:      id.NewUserFromUint(1, t),
		Message:        grp.NewInt(42), // Meaning of Life
		AssociatedData: grp.NewInt(1),
		CurrentKey:     grp.NewInt(1),
	})
	N1RTDecrypt.InChannel <- &inputMsg
	rtnRT := <-N2RTPeel.OutChannel
	esRT := (*rtnRT).(*realtime.Slot)
	t.Logf("RTPEEL:\n  EncryptedMessage: %s\n",
		esRT.Message.Text(10))
	expectedRTPeel := []*cyclic.Int{
		grp.NewInt(42),
	}
	if esRT.Message.Cmp(expectedRTPeel[0]) != 0 {
		t.Errorf("RTPEEL failed EncryptedMessage. Got: %s Expected: %s",
			esRT.Message.Text(10), expectedRTPeel[0].Text(10))
	}

	t.Logf("Final Results: Slot: %d, Recipient ID: %q, Message: %s\n",
		esRT.Slot, *esRT.CurrentID,
		esRT.Message.Text(10))
}

// Perform an end to end test of the precomputation with batch size 1,
// then use it to send the message through a 2-node system to smoke test
// the cryptographic operations.
func MultiNodeTest(nodeCount int, BatchSize uint64,
	group *cyclic.Group, rounds []*globals.Round,
	inputMsgs []realtime.Slot, expectedOutputs []realtime.Slot,
	t *testing.T) {

	benchmark.MultiNodePrecomp(nodeCount, BatchSize, group, rounds)
	benchmark.MultiNodeRealtime(nodeCount, BatchSize, group, rounds, inputMsgs,
		expectedOutputs)
}

func Test3NodeE2E(t *testing.T) {
	nodeCount := 3
	BatchSize := uint64(1)
	grp := cyclic.NewGroup(large.NewInt(107), large.NewInt(4), large.NewInt(5))
	inputMsgs := make([]realtime.Slot, BatchSize)
	outputMsgs := make([]realtime.Slot, BatchSize)
	for i := uint64(0); i < BatchSize; i++ {
		inputMsgs[i] = realtime.Slot{
			Slot:           i,
			CurrentID:      id.NewUserFromUint(i+1, t),
			Message:        grp.NewInt(42 + int64(i)), // Meaning of Life
			AssociatedData: grp.NewInt(1 + int64(i)),
			CurrentKey:     grp.NewInt(1),
		}
		outputMsgs[i] = realtime.Slot{
			Slot:      i,
			CurrentID: id.NewUserFromUint(i+1, t),
			Message:   grp.NewInt(42 + int64(i)), // Meaning of Life
		}
	}
	rounds := benchmark.GenerateRounds(nodeCount, BatchSize, grp)
	MultiNodeTest(nodeCount, BatchSize, grp, rounds, inputMsgs, outputMsgs, t)
}

func TestRealPrimeE2E(t *testing.T) {
	nodeCount := 5
	BatchSize := uint64(10)

	primeStrng := "FFFFFFFFFFFFFFFFC90FDAA22168C234C4C6628B80DC1CD1" +
		"29024E088A67CC74020BBEA63B139B22514A08798E3404DD" +
		"EF9519B3CD3A431B302B0A6DF25F14374FE1356D6D51C245" +
		"E485B576625E7EC6F44C42E9A637ED6B0BFF5CB6F406B7ED" +
		"EE386BFB5A899FA5AE9F24117C4B1FE649286651ECE45B3D" +
		"C2007CB8A163BF0598DA48361C55D39A69163FA8FD24CF5F" +
		"83655D23DCA3AD961C62F356208552BB9ED529077096966D" +
		"670C354E4ABC9804F1746C08CA18217C32905E462E36CE3B" +
		"E39E772C180E86039B2783A2EC07A28FB5C55DF06F4C52C9" +
		"DE2BCBF6955817183995497CEA956AE515D2261898FA0510" +
		"15728E5A8AAAC42DAD33170D04507A33A85521ABDF1CBA64" +
		"ECFB850458DBEF0A8AEA71575D060C7DB3970F85A6E1E4C7" +
		"ABF5AE8CDB0933D71E8C94E04A25619DCEE3D2261AD2EE6B" +
		"F12FFA06D98A0864D87602733EC86A64521F2B18177B200C" +
		"BBE117577A615D6C770988C0BAD946E208E24FA074E5AB31" +
		"43DB5BFCE0FD108E4B82D120A92108011A723C12A787E6D7" +
		"88719A10BDBA5B2699C327186AF4E23C1A946834B6150BDA" +
		"2583E9CA2AD44CE8DBBBC2DB04DE8EF92E8EFC141FBECAA6" +
		"287C59474E6BC05D99B2964FA090C3A2233BA186515BE7ED" +
		"1F612970CEE2D7AFB81BDD762170481CD0069127D5B05AA9" +
		"93B4EA988D8FDDC186FFB7DC90A6C08F4DF435C934063199" +
		"FFFFFFFFFFFFFFFF"

	prime := large.NewInt(0)
	prime.SetString(primeStrng, 16)

	grp := cyclic.NewGroup(prime, large.NewInt(4), large.NewInt(5))
	inputMsgs := make([]realtime.Slot, BatchSize)
	outputMsgs := make([]realtime.Slot, BatchSize)
	for i := uint64(0); i < BatchSize; i++ {
		inputMsgs[i] = realtime.Slot{
			Slot:           i,
			CurrentID:      id.NewUserFromUint(i+1, t),
			Message:        grp.NewInt((42 + int64(i)) % 107), // Meaning of Life
			AssociatedData: grp.NewInt((1 + int64(i)) % 107),
			CurrentKey:     grp.NewInt(1),
		}
		outputMsgs[i] = realtime.Slot{
			Slot:      i,
			CurrentID: id.NewUserFromUint((i+1)%107, t),
			Message:   grp.NewInt((42 + int64(i)) % 107), // Meaning of Life
		}
	}
	rounds := benchmark.GenerateRounds(nodeCount, BatchSize, grp)
	for i := 0; i < nodeCount; i++ {
		for j := uint64(0); j < BatchSize; j++ {
			// Shift by 1
			newj := (j + 1) % BatchSize
			rounds[i].Permutations[j] = newj
		}
		// Now apply  permutations list to outputMsgs
		newOutMsgs := make([]realtime.Slot, BatchSize)
		for j := uint64(0); j < BatchSize; j++ {
			newOutMsgs[rounds[i].Permutations[j]] = outputMsgs[j]
		}
		outputMsgs = newOutMsgs
	}

	MultiNodeTest(nodeCount, BatchSize, grp, rounds, inputMsgs, outputMsgs, t)
}

// Call the main benchmark tests so we get coverage for it
func TestBMPrecomp_1_1(b *testing.T)  { benchmark.PrecompIterations(1, 1, 1) }
func TestBMRealtime_1_1(b *testing.T) { benchmark.RealtimeIterations(1, 1, 1) }
*/
