////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package main

import (
	"fmt"
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/server/cryptops/precomputation"
	"gitlab.com/privategrity/server/cryptops/realtime"
	"gitlab.com/privategrity/server/globals"
	"gitlab.com/privategrity/server/services"
	"strconv"
	"testing"
)

// Convert the round object into a string we can print
func RoundText(g *cyclic.Group, n *globals.Round) string {
	outStr := fmt.Sprintf("\tPrime: 101, Generator: %s, CypherPublicKey: %s, "+
		"Z: %s\n", g.G.Text(10), n.CypherPublicKey.Text(10), n.Z.Text(10))
	outStr += fmt.Sprintf("\tPermutations: %v\n", n.Permutations)
	rfmt := "\tRound[%d]: R(%s, %s, %s) S(%s, %s, %s) T(%s, %s, %s) \n" +
		"\t\t  U(%s, %s, %s) V(%s, %s, %s) \n"
	for i := uint64(0); i < n.BatchSize; i++ {
		outStr += fmt.Sprintf(rfmt, i,
			n.R[i].Text(10), n.R_INV[i].Text(10), n.Y_R[i].Text(10),
			n.S[i].Text(10), n.S_INV[i].Text(10), n.Y_S[i].Text(10),
			n.T[i].Text(10), n.T_INV[i].Text(10), n.Y_T[i].Text(10),
			n.U[i].Text(10), n.U_INV[i].Text(10), n.Y_U[i].Text(10),
			n.V[i].Text(10), n.V_INV[i].Text(10), n.Y_V[i].Text(10))
	}
	return outStr
}

// ComputeSingleNodePrecomputation is a helper func to compute what
// the precomputation should be without any sharing computations for a
// single node system. In other words, it multiplies the R, S, T
// keys together for the message precomputation, and it does the same for
// the U, V keys to make the recipient id precomputation.
func ComputeSingleNodePrecomputation(g *cyclic.Group, round *globals.Round) (
	*cyclic.Int, *cyclic.Int) {
	MP := cyclic.NewInt(1)

	g.Mul(MP, round.R_INV[0], MP)
	g.Mul(MP, round.S_INV[0], MP)
	g.Mul(MP, round.T_INV[0], MP)

	RP := cyclic.NewInt(1)

	g.Mul(RP, round.U_INV[0], RP)
	g.Mul(RP, round.V_INV[0], RP)

	return MP, RP

}

// Compute Precomputation for N nodes
// NOTE: This does not handle precomputation under permutation, but it will
//       handle multi-node precomputation checks.
func ComputePrecomputation(g *cyclic.Group, rounds []*globals.Round) (
	*cyclic.Int, *cyclic.Int) {
	MP := cyclic.NewInt(1)
	RP := cyclic.NewInt(1)
	for i := range rounds {
		g.Mul(MP, rounds[i].R_INV[0], MP)
		g.Mul(MP, rounds[i].S_INV[0], MP)
		g.Mul(MP, rounds[i].T_INV[0], MP)

		g.Mul(RP, rounds[i].U_INV[0], RP)
		g.Mul(RP, rounds[i].V_INV[0], RP)
	}
	return MP, RP
}

// End to end test of the mathematical functions required to "share" 1
// key (i.e., R)
func RootingTest(g *cyclic.Group) {

	K1 := cyclic.NewInt(94)

	Z := cyclic.NewInt(11)

	Y1 := cyclic.NewInt(79)

	gZ := cyclic.NewInt(0)

	gY1 := cyclic.NewInt(0)

	MSG := cyclic.NewInt(0)
	CTXT := cyclic.NewInt(0)

	IVS := cyclic.NewInt(0)
	gY1c := cyclic.NewInt(0)

	RSLT := cyclic.NewInt(0)

	g.Exp(g.G, Z, gZ)
	g.RootCoprime(gZ, Z, RSLT)

	fmt.Printf("GENERATOR:\n  Expected: %s, Result: %s,\n",
		g.G.Text(10), RSLT.Text(10))

	g.Exp(g.G, Y1, gY1)
	g.Mul(K1, gY1, MSG)

	g.Exp(g.G, Z, gZ)
	g.Exp(gZ, Y1, CTXT)

	g.RootCoprime(CTXT, Z, gY1c)

	g.Inverse(gY1c, IVS)

	g.Mul(MSG, IVS, RSLT)

	fmt.Printf("ROOT TEST:\n  Expected: %s, Result: %s,\n",
		gY1.Text(10), gY1c.Text(10))

}

// End to end test of the mathematical functions required to "share" 2 keys
// (i.e., UV)
func RootingTestDouble(g *cyclic.Group) {

	K1 := cyclic.NewInt(94)
	K2 := cyclic.NewInt(18)

	Z := cyclic.NewInt(13)

	Y1 := cyclic.NewInt(87)
	Y2 := cyclic.NewInt(79)

	gZ := cyclic.NewInt(0)

	gY1 := cyclic.NewInt(0)
	gY2 := cyclic.NewInt(0)

	K2gY2 := cyclic.NewInt(0)

	gZY1 := cyclic.NewInt(0)
	gZY2 := cyclic.NewInt(0)

	K1gY1 := cyclic.NewInt(0)
	K1K2gY1Y2 := cyclic.NewInt(0)
	CTXT := cyclic.NewInt(0)

	IVS := cyclic.NewInt(0)
	gY1Y2c := cyclic.NewInt(0)

	RSLT := cyclic.NewInt(0)

	K1K2 := cyclic.NewInt(0)

	g.Exp(g.G, Y1, gY1)
	g.Mul(K1, gY1, K1gY1)

	g.Exp(g.G, Y2, gY2)
	g.Mul(K2, gY2, K2gY2)

	g.Mul(K2gY2, K1gY1, K1K2gY1Y2)

	g.Exp(g.G, Z, gZ)

	g.Exp(gZ, Y1, gZY1)
	g.Exp(gZ, Y2, gZY2)

	g.Mul(gZY1, gZY2, CTXT)

	g.RootCoprime(CTXT, Z, gY1Y2c)

	fmt.Printf("ROUND RECIPIENT PRIVATE KEY: %s,\n", gY1Y2c.Text(10))

	g.Inverse(gY1Y2c, IVS)

	g.Mul(K1K2gY1Y2, IVS, RSLT)

	g.Mul(K1, K2, K1K2)

	fmt.Printf("ROOT TEST DOUBLE:\n  Expected: %s, Result: %s,\n",
		RSLT.Text(10), K1K2.Text(10))

}

// End to end test of the mathematical functions required to "share" 3 keys
// (i.e., RST)
func RootingTestTriple(g *cyclic.Group) {

	K1 := cyclic.NewInt(26)
	K2 := cyclic.NewInt(77)
	K3 := cyclic.NewInt(100)

	Z := cyclic.NewInt(13)

	Y1 := cyclic.NewInt(69)
	Y2 := cyclic.NewInt(81)
	Y3 := cyclic.NewInt(13)

	gZ := cyclic.NewInt(0)

	gY1 := cyclic.NewInt(0)
	gY2 := cyclic.NewInt(0)
	gY3 := cyclic.NewInt(0)

	K1gY1 := cyclic.NewInt(0)
	K2gY2 := cyclic.NewInt(0)
	K3gY3 := cyclic.NewInt(0)

	gZY1 := cyclic.NewInt(0)
	gZY2 := cyclic.NewInt(0)
	gZY3 := cyclic.NewInt(0)

	gZY1Y2 := cyclic.NewInt(0)

	K1K2gY1Y2 := cyclic.NewInt(0)
	K1K2K3gY1Y2Y3 := cyclic.NewInt(0)

	CTXT := cyclic.NewInt(0)

	IVS := cyclic.NewInt(0)
	gY1Y2Y3c := cyclic.NewInt(0)

	RSLT := cyclic.NewInt(0)

	K1K2 := cyclic.NewInt(0)
	K1K2K3 := cyclic.NewInt(0)

	g.Exp(g.G, Y1, gY1)
	g.Mul(K1, gY1, K1gY1)

	g.Exp(g.G, Y2, gY2)
	g.Mul(K2, gY2, K2gY2)

	g.Exp(g.G, Y3, gY3)
	g.Mul(K3, gY3, K3gY3)

	g.Mul(K2gY2, K1gY1, K1K2gY1Y2)
	g.Mul(K1K2gY1Y2, K3gY3, K1K2K3gY1Y2Y3)

	g.Exp(g.G, Z, gZ)

	g.Exp(gZ, Y1, gZY1)
	g.Exp(gZ, Y2, gZY2)
	g.Exp(gZ, Y3, gZY3)

	g.Mul(gZY1, gZY2, gZY1Y2)
	g.Mul(gZY1Y2, gZY3, CTXT)

	g.RootCoprime(CTXT, Z, gY1Y2Y3c)

	g.Inverse(gY1Y2Y3c, IVS)

	g.Mul(K1K2K3gY1Y2Y3, IVS, RSLT)

	g.Mul(K1, K2, K1K2)
	g.Mul(K1K2, K3, K1K2K3)

	fmt.Printf("ROOT TEST TRIPLE:\n  Expected: %s, Result: %s,\n",
		RSLT.Text(10), K1K2K3.Text(10))
}

// Perform an end to end test of the precomputation with batchsize 1,
// then use it to send the message through a 1-node system to smoke test
// the cryptographic operations.
func TestEndToEndCryptops(t *testing.T) {

	// Init, we use a small prime to make it easier to run the numbers
	// when debugging
	batchSize := uint64(1)
	rng := cyclic.NewRandom(cyclic.NewInt(0), cyclic.NewInt(1000))
	grp := cyclic.NewGroup(cyclic.NewInt(101), cyclic.NewInt(5), cyclic.NewInt(4),
		rng)
	round := globals.NewRound(batchSize)
	round.CypherPublicKey = cyclic.NewInt(3)

	// We are the last node, so allocate the arrays for LastNode
	globals.InitLastNode(round)

	// ----- PRECOMPUTATION ----- //

	// GENERATION PHASE
	Generation := services.DispatchCryptop(&grp, precomputation.Generation{},
		nil, nil, round)

	var inMessages []services.Slot
	inMessages = append(inMessages, &precomputation.SlotGeneration{Slot: 0})

	// Kick off processing for generation. This does allocations we need.
	Generation.InChannel <- &(inMessages[0])
	_ = <-Generation.OutChannel

	// These produce useful printouts when the test fails.
	RootingTest(&grp)
	RootingTestDouble(&grp)
	RootingTestTriple(&grp)

	// Overwrite the generated keys. Note the use of Set to make sure the
	// pointers remain unchanged.
	round.Z.Set(cyclic.NewInt(13))

	round.R[0].Set(cyclic.NewInt(35))
	round.R_INV[0].Set(cyclic.NewInt(26))
	round.Y_R[0].Set(cyclic.NewInt(69))

	round.S[0].Set(cyclic.NewInt(21))
	round.S_INV[0].Set(cyclic.NewInt(77))
	round.Y_S[0].Set(cyclic.NewInt(81))

	round.T[0].Set(cyclic.NewInt(100))
	round.T_INV[0].Set(cyclic.NewInt(100))
	round.Y_T[0].Set(cyclic.NewInt(13))

	round.U[0].Set(cyclic.NewInt(72))
	round.U_INV[0].Set(cyclic.NewInt(94))
	round.Y_U[0].Set(cyclic.NewInt(87))

	round.V[0].Set(cyclic.NewInt(73))
	round.V_INV[0].Set(cyclic.NewInt(18))
	round.Y_V[0].Set(cyclic.NewInt(79))

	// SHARE PHASE
	var shareMsg services.Slot
	shareMsg = &precomputation.SlotShare{Slot: 0,
		PartialRoundPublicCypherKey: grp.G}
	Share := services.DispatchCryptop(&grp, precomputation.Share{}, nil, nil,
		round)
	Share.InChannel <- &shareMsg
	shareResultSlot := <-Share.OutChannel
	shareResult := (*shareResultSlot).(*precomputation.SlotShare)
	round.CypherPublicKey.Set(shareResult.PartialRoundPublicCypherKey)

	t.Logf("SHARE:\n")
	t.Logf("%v", RoundText(&grp, round))

	if shareResult.PartialRoundPublicCypherKey.Cmp(cyclic.NewInt(20)) != 0 {
		t.Errorf("SHARE failed, expected 20, got %s",
			shareResult.PartialRoundPublicCypherKey.Text(10))
	}

	// DECRYPT PHASE
	var decMsg services.Slot
	decMsg = &precomputation.PrecomputationSlot{
		Slot:                      0,
		MessageCypher:             cyclic.NewInt(1),
		MessagePrecomputation:     cyclic.NewInt(1),
		RecipientIDCypher:         cyclic.NewInt(1),
		RecipientIDPrecomputation: cyclic.NewInt(1),
	}
	Decrypt := services.DispatchCryptop(&grp, precomputation.Decrypt{},
		nil, nil, round)

	// PERMUTE PHASE
	Permute := services.DispatchCryptop(&grp, precomputation.Permute{},
		nil, nil, round)

	go func(in, out chan *services.Slot) {
		iv := <-in
		is := (*iv).(*precomputation.PrecomputationSlot)

		t.Logf("DECRYPT:\n  MessageCypher: %s, "+
			"RecipientIDCypher: %s,\n"+
			"  MessagePrecomputation: %s, RecipientIDPrecomputation: %s\n",
			is.MessageCypher.Text(10), is.RecipientIDCypher.Text(10),
			is.MessagePrecomputation.Text(10),
			is.RecipientIDPrecomputation.Text(10))

		expectedDecrypt := []*cyclic.Int{
			cyclic.NewInt(32), cyclic.NewInt(35),
			cyclic.NewInt(30), cyclic.NewInt(45),
		}
		if is.MessageCypher.Cmp(expectedDecrypt[0]) != 0 {
			t.Errorf("DECRYPT failed MessageCypher. Got: %s Expected: %s",
				is.MessageCypher.Text(10), expectedDecrypt[0].Text(10))
		}
		if is.RecipientIDCypher.Cmp(expectedDecrypt[1]) != 0 {
			t.Errorf("DECRYPT failed RecipientIDCypher. Got: %s Expected: %s",
				is.RecipientIDCypher.Text(10), expectedDecrypt[1].Text(10))
		}
		if is.MessagePrecomputation.Cmp(expectedDecrypt[2]) != 0 {
			t.Errorf("DECRYPT failed MessagePrecomputation. Got: %s Expected: %s",
				is.MessagePrecomputation.Text(10), expectedDecrypt[2].Text(10))
		}
		if is.RecipientIDPrecomputation.Cmp(expectedDecrypt[3]) != 0 {
			t.Errorf("DECRYPT failed RecipientIDPrecomputation. Got: %s "+
				"Expected: %s", is.RecipientIDPrecomputation.Text(10),
				expectedDecrypt[3].Text(10))
		}

		ov := services.Slot(is)
		out <- &ov
	}(Decrypt.OutChannel, Permute.InChannel)

	// // ENCRYPT PHASE
	Encrypt := services.DispatchCryptop(&grp, precomputation.Encrypt{},
		nil, nil, round)

	go func(in, out chan *services.Slot) {
		iv := <-in
		pm := (*iv).(*precomputation.PrecomputationSlot)

		t.Logf("PERMUTE:\n  MessageCypher: %s, "+
			"RecipientIDCypher: %s,\n"+
			"  MessagePrecomputation: %s, RecipientIDPrecomputation: %s\n",
			pm.MessageCypher.Text(10), pm.RecipientIDCypher.Text(10),
			pm.MessagePrecomputation.Text(10),
			pm.RecipientIDPrecomputation.Text(10))

		expectedPermute := []*cyclic.Int{
			cyclic.NewInt(83), cyclic.NewInt(17),
			cyclic.NewInt(1), cyclic.NewInt(88),
		}
		if pm.MessageCypher.Cmp(expectedPermute[0]) != 0 {
			t.Errorf("PERMUTE failed MessageCypher. Got: %s Expected: %s",
				pm.MessageCypher.Text(10), expectedPermute[0].Text(10))
		}
		if pm.RecipientIDCypher.Cmp(expectedPermute[1]) != 0 {
			t.Errorf("PERMUTE failed RecipientIDCypher. Got: %s Expected: %s",
				pm.RecipientIDCypher.Text(10), expectedPermute[1].Text(10))
		}
		if pm.MessagePrecomputation.Cmp(expectedPermute[2]) != 0 {
			t.Errorf("PERMUTE failed MessagePrecomputation. Got: %s Expected: %s",
				pm.MessagePrecomputation.Text(10), expectedPermute[2].Text(10))
		}
		if pm.RecipientIDPrecomputation.Cmp(expectedPermute[3]) != 0 {
			t.Errorf("PERMUTE failed RecipientIDPrecomputation. Got: %s "+
				"Expected: %s", pm.RecipientIDPrecomputation.Text(10),
				expectedPermute[3].Text(10))
		}

		// Save the results to LastNode, which we don't have to check
		// because we are the only node
		i := pm.Slot
		round.LastNode.RecipientCypherText[i].Set(pm.RecipientIDPrecomputation)
		round.LastNode.EncryptedRecipientPrecomputation[i].Set(
			pm.RecipientIDCypher)

		ov := services.Slot(pm)
		out <- &ov
	}(Permute.OutChannel, Encrypt.InChannel)

	// REVEAL PHASE
	Reveal := services.DispatchCryptop(&grp, precomputation.Reveal{},
		nil, nil, round)

	go func(in, out chan *services.Slot) {
		iv := <-in
		pm := (*iv).(*precomputation.PrecomputationSlot)
		i := pm.Slot

		t.Logf("ENCRYPT:\n  MessageCypher: %s, "+
			"MessagePrecomputation: %s\n", pm.MessageCypher.Text(10),
			pm.MessagePrecomputation.Text(10))

		expectedEncrypt := []*cyclic.Int{
			cyclic.NewInt(57), cyclic.NewInt(9),
		}
		if pm.MessageCypher.Cmp(expectedEncrypt[0]) != 0 {
			t.Errorf("ENCRYPT failed MessageCypher. Got: %s Expected: %s",
				pm.MessageCypher.Text(10), expectedEncrypt[0].Text(10))
		}
		if pm.MessagePrecomputation.Cmp(expectedEncrypt[1]) != 0 {
			t.Errorf("ENCRYPT failed RecipientIDCypher. Got: %s Expected: %s",
				pm.MessagePrecomputation.Text(10), expectedEncrypt[1].Text(10))
		}

		// Save the results to LastNode
		round.LastNode.EncryptedMessagePrecomputation[i].Set(
			pm.MessageCypher)
		ov := services.Slot(pm)
		out <- &ov
	}(Encrypt.OutChannel, Reveal.InChannel)

	// STRIP PHASE
	Strip := services.DispatchCryptop(&grp, precomputation.Strip{},
		nil, nil, round)

	go func(in, out chan *services.Slot) {
		iv := <-in
		pm := (*iv).(*precomputation.PrecomputationSlot)

		t.Logf("REVEAL:\n  RoundMessagePrivateKey: %s, "+
			"RoundRecipientPrivateKey: %s\n", pm.RecipientIDPrecomputation.Text(10),
			pm.RecipientIDPrecomputation.Text(10))
		expectedReveal := []*cyclic.Int{
			cyclic.NewInt(20), cyclic.NewInt(68),
		}
		if pm.MessagePrecomputation.Cmp(expectedReveal[0]) != 0 {
			t.Errorf("REVEAL failed RoundMessagePrivateKey. Got: %s Expected: %s",
				pm.MessagePrecomputation.Text(10), expectedReveal[0].Text(10))
		}
		if pm.RecipientIDPrecomputation.Cmp(expectedReveal[1]) != 0 {
			t.Errorf("REVEAL failed RoundRecipientPrivateKey. Got: %s Expected: %s",
				pm.RecipientIDPrecomputation.Text(10), expectedReveal[1].Text(10))
		}

		ov := services.Slot(pm)
		out <- &ov
	}(Reveal.OutChannel, Strip.InChannel)

	// KICK OFF PRECOMPUTATION and save
	Decrypt.InChannel <- &decMsg
	rtn := <-Strip.OutChannel
	es := (*rtn).(*precomputation.PrecomputationSlot)

	round.LastNode.MessagePrecomputation[es.Slot] = es.MessagePrecomputation
	round.LastNode.RecipientPrecomputation[es.Slot] = es.RecipientIDPrecomputation

	t.Logf("STRIP:\n  MessagePrecomputation: %s, "+
		"RecipientPrecomputation: %s\n", es.MessagePrecomputation.Text(10),
		es.RecipientIDPrecomputation.Text(10))
	expectedStrip := []*cyclic.Int{
		cyclic.NewInt(18), cyclic.NewInt(76),
	}
	if es.MessagePrecomputation.Cmp(expectedStrip[0]) != 0 {
		t.Errorf("STRIP failed MessagePrecomputation. Got: %s Expected: %s",
			es.MessagePrecomputation.Text(10), expectedStrip[0].Text(10))
	}
	if es.RecipientIDPrecomputation.Cmp(expectedStrip[1]) != 0 {
		t.Errorf("STRIP failed RecipientPrecomputation. Got: %s Expected: %s",
			es.RecipientIDPrecomputation.Text(10), expectedStrip[1].Text(10))
	}

	MP, RP := ComputeSingleNodePrecomputation(&grp, round)

	if MP.Cmp(es.MessagePrecomputation) != 0 {
		t.Errorf("Message Precomputation Incorrect! Expected: %s, Received: %s\n",
			MP.Text(10), es.MessagePrecomputation.Text(10))
	}

	if RP.Cmp(es.RecipientIDPrecomputation) != 0 {
		t.Errorf("Recipient Precomputation Incorrect! Expected: %s, Received: %s\n",
			RP.Text(10), es.RecipientIDPrecomputation.Text(10))
	}

	// ----- REALTIME ----- //
	inputMsg := services.Slot(&realtime.RealtimeSlot{
		Slot:               0,
		CurrentID:          1,
		Message:            cyclic.NewInt(31),
		EncryptedRecipient: cyclic.NewInt(1),
		CurrentKey:         cyclic.NewInt(1),
	})

	// DECRYPT PHASE
	RTDecrypt := services.DispatchCryptop(&grp, realtime.Decrypt{},
		nil, nil, round)

	// PERMUTE PHASE
	RTPermute := services.DispatchCryptop(&grp, realtime.Permute{},
		nil, nil, round)

	go func(in, out chan *services.Slot) {
		iv := <-in
		is := (*iv).(*realtime.RealtimeSlot)
		ov := services.Slot(&realtime.RealtimeSlot{
			Slot:               is.Slot,
			Message:            is.Message,
			EncryptedRecipient: is.EncryptedRecipient,
		})

		t.Logf("RTDECRYPT:\n  EncryptedMessage: %s, EncryptedRecipientID: %s\n",
			is.Message.Text(10),
			is.EncryptedRecipient.Text(10))
		expectedRTDecrypt := []*cyclic.Int{
			cyclic.NewInt(75), cyclic.NewInt(72),
		}
		if is.Message.Cmp(expectedRTDecrypt[0]) != 0 {
			t.Errorf("RTDECRYPT failed EncryptedMessage. Got: %s Expected: %s",
				is.Message.Text(10), expectedRTDecrypt[0].Text(10))
		}
		if is.EncryptedRecipient.Cmp(expectedRTDecrypt[1]) != 0 {
			t.Errorf("RTDECRYPT failed EncryptedRecipientID. Got: %s Expected: %s",
				is.EncryptedRecipient.Text(10), expectedRTDecrypt[1].Text(10))
		}

		out <- &ov
	}(RTDecrypt.OutChannel, RTPermute.InChannel)

	// IDENTIFY PHASE
	RTIdentify := services.DispatchCryptop(&grp, realtime.Identify{},
		nil, nil, round)

	RTDecrypt.InChannel <- &inputMsg
	rtnPrm := <-RTPermute.OutChannel
	esPrm := (*rtnPrm).(*realtime.RealtimeSlot)
	ovPrm := services.Slot(&realtime.RealtimeSlot{
		Slot:               esPrm.Slot,
		EncryptedRecipient: esPrm.EncryptedRecipient,
	})
	TmpMsg := esPrm.Message
	t.Logf("RTPERMUTE:\n  EncryptedRecipientID: %s\n",
		esPrm.EncryptedRecipient.Text(10))
	expectedRTPermute := []*cyclic.Int{
		cyclic.NewInt(4),
	}
	if esPrm.EncryptedRecipient.Cmp(expectedRTPermute[0]) != 0 {
		t.Errorf("RTPERMUTE failed EncryptedRecipientID. Got: %s Expected: %s",
			esPrm.EncryptedRecipient.Text(10), expectedRTPermute[0].Text(10))
	}

	RTIdentify.InChannel <- &ovPrm
	rtnTmp := <-RTIdentify.OutChannel
	esTmp := (*rtnTmp).(*realtime.RealtimeSlot)
	rID, _ := strconv.ParseUint(esTmp.EncryptedRecipient.Text(10), 10, 64)
	inputMsgPostID := services.Slot(&realtime.RealtimeSlot{
		Slot:       esTmp.Slot,
		CurrentID:  rID,
		Message:    TmpMsg,
		CurrentKey: cyclic.NewInt(1),
	})
	t.Logf("RTIDENTIFY:\n  RecipientID: %s\n",
		esTmp.EncryptedRecipient.Text(10))
	expectedRTIdentify := []*cyclic.Int{
		cyclic.NewInt(1),
	}
	if esTmp.EncryptedRecipient.Cmp(expectedRTIdentify[0]) != 0 {
		t.Errorf("RTIDENTIFY failed EncryptedRecipientID. Got: %s Expected: %s",
			esTmp.EncryptedRecipient.Text(10), expectedRTIdentify[0].Text(10))
	}

	// ENCRYPT PHASE
	RTEncrypt := services.DispatchCryptop(&grp, realtime.Encrypt{},
		nil, nil, round)

	// PEEL PHASE
	RTPeel := services.DispatchCryptop(&grp, realtime.Peel{},
		nil, nil, round)

	go func(in, out chan *services.Slot) {
		iv := <-in
		is := realtime.RealtimeSlot(*((*iv).(*realtime.RealtimeSlot)))
		ov := services.Slot(&is)

		t.Logf("RTENCRYPT:\n  EncryptedMessage: %s\n",
			is.Message.Text(10))
		expectedRTEncrypt := []*cyclic.Int{
			cyclic.NewInt(41),
		}
		if is.Message.Cmp(expectedRTEncrypt[0]) != 0 {
			t.Errorf("RTENCRYPT failed EncryptedMessage. Got: %s Expected: %s",
				is.Message.Text(10), expectedRTEncrypt[0].Text(10))
		}

		out <- &ov
	}(RTEncrypt.OutChannel, RTPeel.InChannel)

	// KICK OFF RT COMPUTATION
	RTEncrypt.InChannel <- &inputMsgPostID
	rtnRT := <-RTPeel.OutChannel
	esRT := (*rtnRT).(*realtime.RealtimeSlot)

	t.Logf("RTPEEL:\n  EncryptedMessage: %s\n",
		esRT.Message.Text(10))
	expectedRTPeel := []*cyclic.Int{
		cyclic.NewInt(31),
	}
	if esRT.Message.Cmp(expectedRTPeel[0]) != 0 {
		t.Errorf("RTPEEL failed EncryptedMessage. Got: %s Expected: %s",
			esRT.Message.Text(10), expectedRTPeel[0].Text(10))
	}

	t.Logf("Final Results: Slot: %d, Recipient ID: %d, Message: %s\n",
		esRT.Slot, esRT.CurrentID,
		esRT.Message.Text(10))
}

// Convert Decrypt output slot to Permute input slot
func DecryptPermuteTranslate(decrypt, permute chan *services.Slot) {
	for decryptSlot := range decrypt {
		is := (*decryptSlot).(*precomputation.PrecomputationSlot)
		sp := services.Slot(is)
		permute <- &sp
	}
}

// Convert Permute output slot to Encrypt input slot
func PermuteEncryptTranslate(permute, encrypt chan *services.Slot,
	round *globals.Round) {
	for permuteSlot := range permute {
		is := (*permuteSlot).(*precomputation.PrecomputationSlot)
		se := services.Slot(&precomputation.PrecomputationSlot{
			Slot:                  is.Slot,
			MessageCypher:         is.MessageCypher,
			MessagePrecomputation: is.MessagePrecomputation,
		})
		// Save LastNode Data to Round
		i := is.Slot
		round.LastNode.RecipientCypherText[i].Set(is.RecipientIDPrecomputation)
		round.LastNode.EncryptedRecipientPrecomputation[i].Set(
			is.RecipientIDCypher)
		encrypt <- &se
	}
}

// Convert Encrypt output slot to Reveal input slot
func EncryptRevealTranslate(encrypt, reveal chan *services.Slot,
	round *globals.Round) {
	for encryptSlot := range encrypt {
		is := (*encryptSlot).(*precomputation.PrecomputationSlot)
		i := is.Slot
		sr := services.Slot(&precomputation.PrecomputationSlot{
			Slot: i,
			MessagePrecomputation:     is.MessagePrecomputation,
			RecipientIDPrecomputation: round.LastNode.RecipientCypherText[i],
		})
		round.LastNode.EncryptedMessagePrecomputation[i].Set(
			is.MessageCypher)
		reveal <- &sr
	}
}

// Convert Reveal output slot to Strip input slot
func RevealStripTranslate(reveal, strip chan *services.Slot) {
	for revealSlot := range reveal {
		is := (*revealSlot).(*precomputation.PrecomputationSlot)
		i := is.Slot
		ss := services.Slot(&precomputation.PrecomputationSlot{
			Slot: i,
			MessagePrecomputation:     is.MessagePrecomputation,
			RecipientIDPrecomputation: is.RecipientIDPrecomputation,
		})
		strip <- &ss
	}
}

// Convert RTDecrypt output slot to RTPermute input slot
func RTDecryptRTPermuteTranslate(decrypt, permute chan *services.Slot) {
	for decryptSlot := range decrypt {
		is := (*decryptSlot).(*realtime.RealtimeSlot)
		ov := services.Slot(&realtime.RealtimeSlot{
			Slot:               is.Slot,
			Message:            is.Message,
			EncryptedRecipient: is.EncryptedRecipient,
		})
		permute <- &ov
	}
}

func RTPermuteRTIdentifyTranslate(permute, identify chan *services.Slot,
	outMsgs []*cyclic.Int) {
	for permuteSlot := range permute {
		esPrm := (*permuteSlot).(*realtime.RealtimeSlot)
		ovPrm := services.Slot(&realtime.RealtimeSlot{
			Slot:               esPrm.Slot,
			EncryptedRecipient: esPrm.EncryptedRecipient,
		})
		outMsgs[esPrm.Slot].Set(esPrm.Message)
		identify <- &ovPrm
	}
}

func RTIdentifyRTEncryptTranslate(identify, encrypt chan *services.Slot,
	inMsgs []*cyclic.Int) {
	for identifySlot := range identify {
		esTmp := (*identifySlot).(*realtime.RealtimeSlot)
		rID, _ := strconv.ParseUint(esTmp.EncryptedRecipient.Text(10), 10, 64)
		inputMsgPostID := services.Slot(&realtime.RealtimeSlot{
			Slot:       esTmp.Slot,
			CurrentID:  rID,
			Message:    inMsgs[esTmp.Slot],
			CurrentKey: cyclic.NewInt(1),
		})
		encrypt <- &inputMsgPostID
	}
}

func RTEncryptRTPeelTranslate(encrypt, peel chan *services.Slot) {
	for encryptSlot := range encrypt {
		is := realtime.RealtimeSlot(*((*encryptSlot).(*realtime.RealtimeSlot)))
		ov := services.Slot(&is)
		peel <- &ov
	}
}

func RTDecryptRTDecryptTranslate(in, out chan *services.Slot) {
	for is := range in {
		o := (*is).(*realtime.RealtimeSlot)
		os := services.Slot(&realtime.RealtimeSlot{
			Slot:               o.Slot,
			CurrentID:          o.CurrentID,
			Message:            o.Message,
			EncryptedRecipient: o.EncryptedRecipient,
			CurrentKey:         cyclic.NewInt(1), // WTF? FIXME
		})
		out <- &os
	}
}

func RTEncryptRTEncryptTranslate(in, out chan *services.Slot) {
	for is := range in {
		o := (*is).(*realtime.RealtimeSlot)
		os := services.Slot(&realtime.RealtimeSlot{
			Slot:       o.Slot,
			CurrentID:  o.CurrentID,
			Message:    o.Message,
			CurrentKey: cyclic.NewInt(1), // FIXME
		})
		out <- &os
	}
}

// Perform an end to end test of the precomputation with batchsize 1,
// then use it to send the message through a 2-node system to smoke test
// the cryptographic operations.
func TestEndToEndCryptopsWith2Nodes(t *testing.T) {

	// Init, we use a small prime to make it easier to run the numbers
	// when debugging
	batchSize := uint64(1)
	rng := cyclic.NewRandom(cyclic.NewInt(0), cyclic.NewInt(1000))
	grp := cyclic.NewGroup(cyclic.NewInt(101), cyclic.NewInt(5), cyclic.NewInt(4),
		rng)
	Node1Round := globals.NewRound(batchSize)
	Node2Round := globals.NewRound(batchSize)
	Node1Round.CypherPublicKey = cyclic.NewInt(0)
	Node2Round.CypherPublicKey = cyclic.NewInt(0)

	// Allocate the arrays for LastNode
	globals.InitLastNode(Node2Round)

	// ----- PRECOMPUTATION ----- //
	N1Generation := services.DispatchCryptop(&grp, precomputation.Generation{},
		nil, nil, Node1Round)
	N2Generation := services.DispatchCryptop(&grp, precomputation.Generation{},
		nil, nil, Node2Round)

	N1Share := services.DispatchCryptop(&grp, precomputation.Share{}, nil, nil,
		Node1Round)
	N2Share := services.DispatchCryptop(&grp, precomputation.Share{},
		N1Share.OutChannel, nil, Node2Round)

	N1Decrypt := services.DispatchCryptop(&grp, precomputation.Decrypt{},
		nil, nil, Node1Round)
	N2Decrypt := services.DispatchCryptop(&grp, precomputation.Decrypt{},
		N1Decrypt.OutChannel, nil, Node2Round)

	N1Permute := services.DispatchCryptop(&grp, precomputation.Permute{},
		N2Decrypt.OutChannel, nil, Node1Round)
	N2Permute := services.DispatchCryptop(&grp, precomputation.Permute{},
		N1Permute.OutChannel, nil, Node2Round)

	N1Encrypt := services.DispatchCryptop(&grp, precomputation.Encrypt{},
		nil, nil, Node1Round)
	N2Encrypt := services.DispatchCryptop(&grp, precomputation.Encrypt{},
		N1Encrypt.OutChannel, nil, Node2Round)

	N1Reveal := services.DispatchCryptop(&grp, precomputation.Reveal{},
		nil, nil, Node1Round)
	N2Reveal := services.DispatchCryptop(&grp, precomputation.Reveal{},
		N1Reveal.OutChannel, nil, Node2Round)

	N2Strip := services.DispatchCryptop(&grp, precomputation.Strip{},
		nil, nil, Node2Round)

	go RevealStripTranslate(N2Reveal.OutChannel, N2Strip.InChannel)
	go EncryptRevealTranslate(N2Encrypt.OutChannel, N1Reveal.InChannel,
		Node2Round)
	go PermuteEncryptTranslate(N2Permute.OutChannel, N1Encrypt.InChannel,
		Node2Round)

	// Run Generate
	genMsg := services.Slot(&precomputation.SlotGeneration{Slot: 0})
	N1Generation.InChannel <- &genMsg
	_ = <-N1Generation.OutChannel
	N2Generation.InChannel <- &genMsg
	_ = <-N2Generation.OutChannel

	t.Logf("2 NODE GENERATION RESULTS: \n")
	t.Logf("%v", RoundText(&grp, Node1Round))
	t.Logf("%v", RoundText(&grp, Node2Round))

	// TODO: Pre-can the keys to use here if necessary.

	// Run Share -- Then save the result to both rounds
	// Note that the outchannel for N1Share is the input channel for N2share
	shareMsg := services.Slot(&precomputation.SlotShare{
		PartialRoundPublicCypherKey: grp.G})
	N1Share.InChannel <- &shareMsg
	shareResultSlot := <-N2Share.OutChannel
	shareResult := (*shareResultSlot).(*precomputation.SlotShare)
	PublicCypherKey := cyclic.NewInt(0)
	PublicCypherKey.Set(shareResult.PartialRoundPublicCypherKey)
	Node1Round.CypherPublicKey.Set(PublicCypherKey)
	Node2Round.CypherPublicKey.Set(PublicCypherKey)

	t.Logf("2 NODE SHARE RESULTS: \n")
	t.Logf("%v", RoundText(&grp, Node2Round))
	t.Logf("%v", RoundText(&grp, Node1Round))

	// Now finish precomputation
	decMsg := services.Slot(&precomputation.PrecomputationSlot{
		Slot:                      0,
		MessageCypher:             cyclic.NewInt(1),
		MessagePrecomputation:     cyclic.NewInt(1),
		RecipientIDCypher:         cyclic.NewInt(1),
		RecipientIDPrecomputation: cyclic.NewInt(1),
	})
	N1Decrypt.InChannel <- &decMsg
	rtn := <-N2Strip.OutChannel
	es := (*rtn).(*precomputation.PrecomputationSlot)

	Node2Round.LastNode.MessagePrecomputation[es.Slot] = es.MessagePrecomputation
	Node2Round.LastNode.RecipientPrecomputation[es.Slot] =
		es.RecipientIDPrecomputation
	t.Logf("2 NODE STRIP:\n  MessagePrecomputation: %s, "+
		"RecipientPrecomputation: %s\n", es.MessagePrecomputation.Text(10),
		es.RecipientIDPrecomputation.Text(10))

	// Check precomputation
	MP, RP := ComputePrecomputation(&grp,
		[]*globals.Round{Node1Round, Node2Round})

	if MP.Cmp(es.MessagePrecomputation) != 0 {
		t.Errorf("Message Precomputation Incorrect! Expected: %s, Received: %s\n",
			MP.Text(10), es.MessagePrecomputation.Text(10))
	}
	if RP.Cmp(es.RecipientIDPrecomputation) != 0 {
		t.Errorf("Recipient Precomputation Incorrect! Expected: %s, Received: %s\n",
			RP.Text(10), es.RecipientIDPrecomputation.Text(10))
	}

	// ----- REALTIME ----- //
	IntermediateMsgs := make([]*cyclic.Int, 1)
	IntermediateMsgs[0] = cyclic.NewInt(0)

	N1RTDecrypt := services.DispatchCryptop(&grp, realtime.Decrypt{},
		nil, nil, Node1Round)
	N2RTDecrypt := services.DispatchCryptop(&grp, realtime.Decrypt{},
		nil, nil, Node2Round)

	N1RTPermute := services.DispatchCryptop(&grp, realtime.Permute{},
		nil, nil, Node1Round)
	N2RTPermute := services.DispatchCryptop(&grp, realtime.Permute{},
		N1RTPermute.OutChannel, nil, Node2Round)

	N2RTIdentify := services.DispatchCryptop(&grp, realtime.Identify{},
		nil, nil, Node2Round)

	N1RTEncrypt := services.DispatchCryptop(&grp, realtime.Encrypt{},
		nil, nil, Node1Round)
	N2RTEncrypt := services.DispatchCryptop(&grp, realtime.Encrypt{},
		nil, nil, Node2Round)

	N2RTPeel := services.DispatchCryptop(&grp, realtime.Peel{},
		nil, nil, Node2Round)

	go RTEncryptRTEncryptTranslate(N1RTEncrypt.OutChannel, N2RTEncrypt.InChannel)
	go RTDecryptRTDecryptTranslate(N1RTDecrypt.OutChannel, N2RTDecrypt.InChannel)
	go RTDecryptRTPermuteTranslate(N2RTDecrypt.OutChannel, N1RTPermute.InChannel)
	go RTPermuteRTIdentifyTranslate(N2RTPermute.OutChannel,
		N2RTIdentify.InChannel, IntermediateMsgs)
	go RTIdentifyRTEncryptTranslate(N2RTIdentify.OutChannel,
		N1RTEncrypt.InChannel, IntermediateMsgs)
	go RTEncryptRTPeelTranslate(N2RTEncrypt.OutChannel, N2RTPeel.InChannel)

	inputMsg := services.Slot(&realtime.RealtimeSlot{
		Slot:               0,
		CurrentID:          1,
		Message:            cyclic.NewInt(42), // Meaning of Life
		EncryptedRecipient: cyclic.NewInt(1),
		CurrentKey:         cyclic.NewInt(1),
	})
	N1RTDecrypt.InChannel <- &inputMsg
	rtnRT := <-N2RTPeel.OutChannel
	esRT := (*rtnRT).(*realtime.RealtimeSlot)
	t.Logf("RTPEEL:\n  EncryptedMessage: %s\n",
		esRT.Message.Text(10))
	expectedRTPeel := []*cyclic.Int{
		cyclic.NewInt(42),
	}
	if esRT.Message.Cmp(expectedRTPeel[0]) != 0 {
		t.Errorf("RTPEEL failed EncryptedMessage. Got: %s Expected: %s",
			esRT.Message.Text(10), expectedRTPeel[0].Text(10))
	}

	t.Logf("Final Results: Slot: %d, Recipient ID: %d, Message: %s\n",
		esRT.Slot, esRT.CurrentID,
		esRT.Message.Text(10))
}

// Helper function to initialize round keys. Useful when you only need to edit 1
// element (e.g., the Permutation) in the set of keys held in round
func GenerateRounds(nodeCount int, BatchSize uint64,
	group *cyclic.Group, t testing.TB) []*globals.Round {
	rounds := make([]*globals.Round, nodeCount)
	for i := 0; i < nodeCount; i++ {
		rounds[i] = globals.NewRound(BatchSize)
		rounds[i].CypherPublicKey = cyclic.NewInt(0)
		// Last Node initialization
		if i == (nodeCount - 1) {
			globals.InitLastNode(rounds[i])
		}
	}

	// Run the GENERATION step
	generations := make([]*services.ThreadController, nodeCount)
	for i := 0; i < nodeCount; i++ {
		generations[i] = services.DispatchCryptop(group,
			precomputation.Generation{}, nil, nil, rounds[i])
	}
	for i := 0; i < nodeCount; i++ {
		for j := uint64(0); j < BatchSize; j++ {
			genMsg := services.Slot(&precomputation.SlotGeneration{Slot: j})
			generations[i].InChannel <- &genMsg
			_ = <-generations[i].OutChannel
		}
	}

	t.Logf("%d NODE GENERATION RESULTS: \n", nodeCount)
	for i := 0; i < nodeCount; i++ {
		t.Logf("%v", RoundText(group, rounds[i]))
	}

	return rounds
}

func MultiNodePrecomp(nodeCount int, BatchSize uint64,
	group *cyclic.Group, rounds []*globals.Round, t testing.TB) {
	LastRound := rounds[nodeCount-1]

	// ----- PRECOMPUTATION ----- //

	shares := make([]*services.ThreadController, nodeCount)
	decrypts := make([]*services.ThreadController, nodeCount)
	permutes := make([]*services.ThreadController, nodeCount)
	encrypts := make([]*services.ThreadController, nodeCount)
	reveals := make([]*services.ThreadController, nodeCount)

	decrPerm := make(chan *services.Slot)

	for i := 0; i < nodeCount; i++ {
		if i == 0 {
			if nodeCount == 1{
				shares[i] = services.DispatchCryptop(group, precomputation.Share{},
					nil, nil, rounds[i])
				decrypts[i] = services.DispatchCryptop(group, precomputation.Decrypt{},
					nil, decrPerm, rounds[i])
				permutes[i] = services.DispatchCryptop(group, precomputation.Permute{},
					decrPerm, nil, rounds[i])
				encrypts[i] = services.DispatchCryptop(group, precomputation.Encrypt{},
					nil, nil, rounds[i])
				reveals[i] = services.DispatchCryptop(group, precomputation.Reveal{},
					nil, nil, rounds[i])
			}else{
				shares[i] = services.DispatchCryptop(group, precomputation.Share{},
					nil, nil, rounds[i])
				decrypts[i] = services.DispatchCryptop(group, precomputation.Decrypt{},
					nil, nil, rounds[i])
				permutes[i] = services.DispatchCryptop(group, precomputation.Permute{},
					decrPerm, nil, rounds[i])
				encrypts[i] = services.DispatchCryptop(group, precomputation.Encrypt{},
					nil, nil, rounds[i])
				reveals[i] = services.DispatchCryptop(group, precomputation.Reveal{},
					nil, nil, rounds[i])
			}

		} else if i < (nodeCount - 1) {
			shares[i] = services.DispatchCryptop(group, precomputation.Share{},
				shares[i-1].OutChannel, nil, rounds[i])
			decrypts[i] = services.DispatchCryptop(group, precomputation.Decrypt{},
				decrypts[i-1].OutChannel, nil, rounds[i])
			permutes[i] = services.DispatchCryptop(group, precomputation.Permute{},
				permutes[i-1].OutChannel, nil, rounds[i])
			encrypts[i] = services.DispatchCryptop(group, precomputation.Encrypt{},
				encrypts[i-1].OutChannel, nil, rounds[i])
			reveals[i] = services.DispatchCryptop(group, precomputation.Reveal{},
				reveals[i-1].OutChannel, nil, rounds[i])
		} else {
			shares[i] = services.DispatchCryptop(group, precomputation.Share{},
				shares[i-1].OutChannel, nil, rounds[i])
			decrypts[i] = services.DispatchCryptop(group, precomputation.Decrypt{},
				decrypts[i-1].OutChannel, decrPerm, rounds[i])
			permutes[i] = services.DispatchCryptop(group, precomputation.Permute{},
				permutes[i-1].OutChannel, nil, rounds[i])
			encrypts[i] = services.DispatchCryptop(group, precomputation.Encrypt{},
				encrypts[i-1].OutChannel, nil, rounds[i])
			reveals[i] = services.DispatchCryptop(group, precomputation.Reveal{},
				reveals[i-1].OutChannel, nil, rounds[i])
		}
	}



	LNStrip := services.DispatchCryptop(group, precomputation.Strip{},
		nil, nil, LastRound)

	go RevealStripTranslate(reveals[nodeCount-1].OutChannel,
		LNStrip.InChannel)
	go EncryptRevealTranslate(encrypts[nodeCount-1].OutChannel,
		reveals[0].InChannel, LastRound)
	go PermuteEncryptTranslate(permutes[nodeCount-1].OutChannel,
		encrypts[0].InChannel, LastRound)
	//go DecryptPermuteTranslate(decrypts[nodeCount-1].OutChannel,
	//	permutes[0].InChannel)

	// Run Share -- Then save the result to both rounds
	// Note that the outchannel for N1Share is the input channel for N2share
	shareMsg := services.Slot(&precomputation.SlotShare{
		PartialRoundPublicCypherKey: group.G})
	shares[0].InChannel <- &shareMsg
	shareResultSlot := <-shares[nodeCount-1].OutChannel
	shareResult := (*shareResultSlot).(*precomputation.SlotShare)
	PublicCypherKey := cyclic.NewInt(0)
	PublicCypherKey.Set(shareResult.PartialRoundPublicCypherKey)
	for i := 0; i < nodeCount; i++ {
		rounds[i].CypherPublicKey.Set(PublicCypherKey)
	}

	t.Logf("%d NODE SHARE RESULTS: \n", nodeCount)
	for i := 0; i < nodeCount; i++ {
		t.Logf("%v", RoundText(group, rounds[i]))
	}

	// Now finish precomputation
	for i := uint64(0); i < BatchSize; i++ {
		decMsg := services.Slot(&precomputation.PrecomputationSlot{
			Slot:                      i,
			MessageCypher:             cyclic.NewInt(1),
			MessagePrecomputation:     cyclic.NewInt(1),
			RecipientIDCypher:         cyclic.NewInt(1),
			RecipientIDPrecomputation: cyclic.NewInt(1),
		})
		decrypts[0].InChannel <- &decMsg
	}

	for i := uint64(0); i < BatchSize; i++ {
		rtn := <-LNStrip.OutChannel
		es := (*rtn).(*precomputation.PrecomputationSlot)

		LastRound.LastNode.MessagePrecomputation[es.Slot] = es.MessagePrecomputation
		LastRound.LastNode.RecipientPrecomputation[es.Slot] =
			es.RecipientIDPrecomputation

		t.Logf("%d NODE STRIP:\n  MessagePrecomputation: %s, "+
			"RecipientPrecomputation: %s\n", nodeCount,
			es.MessagePrecomputation.Text(10),
			es.RecipientIDPrecomputation.Text(10))

		// Check precomputation, note that these are currently expected to be
		// wrong under permutation
		// MP, RP := ComputePrecomputation(group, rounds)

		// if MP.Cmp(es.MessagePrecomputation) != 0 {
		// 	t.Logf("Message Precomputation Incorrect! Expected: %s, "+
		// 		"Received: %s\n",
		// 		MP.Text(10), es.MessagePrecomputation.Text(10))
		// }
		// if RP.Cmp(es.RecipientPrecomputation) != 0 {
		// 	t.Logf("Recipient Precomputation Incorrect! Expected: %s,"+
		// 		" Received: %s\n",
		// 		RP.Text(10), es.RecipientPrecomputation.Text(10))
		// }
	}
}

func MultiNodeRealtime(nodeCount int, BatchSize uint64,
	group *cyclic.Group, rounds []*globals.Round,
	inputMsgs []realtime.RealtimeSlot, expectedOutputs []realtime.RealtimeSlot,
	t testing.TB) {

	LastRound := rounds[nodeCount-1]

	// ----- REALTIME ----- //
	IntermediateMsgs := make([]*cyclic.Int, BatchSize)
	for i := uint64(0); i < BatchSize; i++ {
		IntermediateMsgs[i] = cyclic.NewInt(0)
	}
	rtdecrypts := make([]*services.ThreadController, nodeCount)
	rtpermutes := make([]*services.ThreadController, nodeCount)
	reorgs := make([]*services.ThreadController, nodeCount)
	rtencrypts := make([]*services.ThreadController, nodeCount)
	for i := 0; i < nodeCount; i++ {
		rtdecrypts[i] = services.DispatchCryptop(group,
			realtime.Decrypt{}, nil, nil, rounds[i])

		// NOTE: Permute -> reorg -> Permute -> ... -> reorg -> Identify
		reorgs[i] = services.NewSlotReorganizer(nil, nil, BatchSize)
		if i == 0 {
			rtpermutes[i] = services.DispatchCryptop(group,
				realtime.Permute{}, nil, reorgs[i].InChannel, rounds[i])
		} else {
			rtpermutes[i] = services.DispatchCryptop(group,
				realtime.Permute{}, reorgs[i-1].OutChannel, reorgs[i].InChannel,
				rounds[i])
		}
		rtencrypts[i] = services.DispatchCryptop(group,
			realtime.Encrypt{}, nil, nil, rounds[i])

		if i != 0 {
			go RTEncryptRTEncryptTranslate(rtencrypts[i-1].OutChannel,
				rtencrypts[i].InChannel)
			go RTDecryptRTDecryptTranslate(rtdecrypts[i-1].OutChannel,
				rtdecrypts[i].InChannel)
		}
	}

	LNRTIdentify := services.DispatchCryptop(group,
		realtime.Identify{}, nil, nil, LastRound)
	LNRTPeel := services.DispatchCryptop(group,
		realtime.Peel{}, nil, nil, LastRound)

	go RTDecryptRTPermuteTranslate(rtdecrypts[nodeCount-1].OutChannel,
		rtpermutes[0].InChannel)
	go RTPermuteRTIdentifyTranslate(reorgs[nodeCount-1].OutChannel,
		LNRTIdentify.InChannel, IntermediateMsgs)
	go RTIdentifyRTEncryptTranslate(LNRTIdentify.OutChannel,
		rtencrypts[0].InChannel, IntermediateMsgs)
	go RTEncryptRTPeelTranslate(rtencrypts[nodeCount-1].OutChannel,
		LNRTPeel.InChannel)

	for i := uint64(0); i < BatchSize; i++ {
		in := services.Slot(&inputMsgs[i])
		rtdecrypts[0].InChannel <- &in
	}

	for i := uint64(0); i < BatchSize; i++ {
		rtnRT := <-LNRTPeel.OutChannel
		esRT := (*rtnRT).(*realtime.RealtimeSlot)
		t.Logf("RTPEEL:\n  EncryptedMessage: %s\n",
			esRT.Message.Text(10))

		if esRT.Message.Cmp(expectedOutputs[i].Message) != 0 {
			t.Errorf("RTPEEL %d failed EncryptedMessage. Got: %s Expected: %s",
				esRT.Slot,
				esRT.Message.Text(10),
				expectedOutputs[i].Message.Text(10))
		}
		if esRT.CurrentID != expectedOutputs[i].CurrentID {
			t.Errorf("RTPEEL %d failed RecipientID. Got: %d Expected: %d",
				esRT.Slot, esRT.CurrentID, expectedOutputs[i].CurrentID)
		}

		t.Logf("Final Results: Slot: %d, Recipient ID: %d, Message: %s\n",
			esRT.Slot, esRT.CurrentID, esRT.Message.Text(10))
	}

}

// Perform an end to end test of the precomputation with batchsize 1,
// then use it to send the message through a 2-node system to smoke test
// the cryptographic operations.
func MultiNodeTest(nodeCount int, BatchSize uint64,
	group *cyclic.Group, rounds []*globals.Round,
	inputMsgs []realtime.RealtimeSlot, expectedOutputs []realtime.RealtimeSlot,
	t *testing.T) {

	MultiNodePrecomp(nodeCount, BatchSize, group, rounds, t)
	MultiNodeRealtime(nodeCount, BatchSize, group, rounds, inputMsgs,
		expectedOutputs, t)
}

func Test3NodeE2E(t *testing.T) {
	nodeCount := 3
	BatchSize := uint64(1)
	rng := cyclic.NewRandom(cyclic.NewInt(0), cyclic.NewInt(1000))
	grp := cyclic.NewGroup(cyclic.NewInt(101), cyclic.NewInt(5), cyclic.NewInt(4),
		rng)
	inputMsgs := make([]realtime.RealtimeSlot, BatchSize)
	outputMsgs := make([]realtime.RealtimeSlot, BatchSize)
	for i := uint64(0); i < BatchSize; i++ {
		inputMsgs[i] = realtime.RealtimeSlot{
			Slot:               i,
			CurrentID:          i + 1,
			Message:            cyclic.NewInt(42 + int64(i)), // Meaning of Life
			EncryptedRecipient: cyclic.NewInt(1 + int64(i)),
			CurrentKey:         cyclic.NewInt(1),
		}
		outputMsgs[i] = realtime.RealtimeSlot{
			Slot:      i,
			CurrentID: i + 1,
			Message:   cyclic.NewInt(42 + int64(i)), // Meaning of Life
		}
	}
	rounds := GenerateRounds(nodeCount, BatchSize, &grp, t)
	MultiNodeTest(nodeCount, BatchSize, &grp, rounds, inputMsgs, outputMsgs, t)
}

func Test1NodePermuteE2E(t *testing.T) {
	nodeCount := 1
	BatchSize := uint64(100)

	primeStrng := "101"

	prime := cyclic.NewInt(0)
	prime.SetString(primeStrng, 10)

	rng := cyclic.NewRandom(cyclic.NewInt(0), cyclic.NewInt(1000))
	grp := cyclic.NewGroup(prime, cyclic.NewInt(5), cyclic.NewInt(4),
		rng)
	inputMsgs := make([]realtime.RealtimeSlot, BatchSize)
	outputMsgs := make([]realtime.RealtimeSlot, BatchSize)
	for i := uint64(0); i < BatchSize; i++ {
		inputMsgs[i] = realtime.RealtimeSlot{
			Slot:               i,
			CurrentID:          i + 1,
			Message:            cyclic.NewInt((42 + int64(i)) % 101), // Meaning of Life
			EncryptedRecipient: cyclic.NewInt((1 + int64(i)) % 101),
			CurrentKey:         cyclic.NewInt(1),
		}
		outputMsgs[i] = realtime.RealtimeSlot{
			Slot:      i,
			CurrentID: (i + 1) % 101,
			Message:   cyclic.NewInt((42 + int64(i)) % 101), // Meaning of Life
		}
	}
	rounds := GenerateRounds(nodeCount, BatchSize, &grp, t)
	for i := 0; i < nodeCount; i++ {
		for j := uint64(0); j < BatchSize; j++ {
			// Shift by 1
			newj := (j + 1) % BatchSize
			rounds[i].Permutations[j] = newj
		}
		// Now apply  permutations list to outputMsgs
		newOutMsgs := make([]realtime.RealtimeSlot, BatchSize)
		for j := uint64(0); j < BatchSize; j++ {
			newOutMsgs[rounds[i].Permutations[j]] = outputMsgs[j]
		}
		outputMsgs = newOutMsgs
	}

	MultiNodeTest(nodeCount, BatchSize, &grp, rounds, inputMsgs, outputMsgs, t)
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

	prime := cyclic.NewInt(0)
	prime.SetString(primeStrng, 16)

	rng := cyclic.NewRandom(cyclic.NewInt(0), cyclic.NewInt(1000))
	grp := cyclic.NewGroup(prime, cyclic.NewInt(5), cyclic.NewInt(4),
		rng)
	inputMsgs := make([]realtime.RealtimeSlot, BatchSize)
	outputMsgs := make([]realtime.RealtimeSlot, BatchSize)
	for i := uint64(0); i < BatchSize; i++ {
		inputMsgs[i] = realtime.RealtimeSlot{
			Slot:               i,
			CurrentID:          i + 1,
			Message:            cyclic.NewInt((42 + int64(i)) % 101), // Meaning of Life
			EncryptedRecipient: cyclic.NewInt((1 + int64(i)) % 101),
			CurrentKey:         cyclic.NewInt(1),
		}
		outputMsgs[i] = realtime.RealtimeSlot{
			Slot:      i,
			CurrentID: (i + 1) % 101,
			Message:   cyclic.NewInt((42 + int64(i)) % 101), // Meaning of Life
		}
	}
	rounds := GenerateRounds(nodeCount, BatchSize, &grp, t)
	for i := 0; i < nodeCount; i++ {
		for j := uint64(0); j < BatchSize; j++ {
			// Shift by 1
			newj := (j + 1) % BatchSize
			rounds[i].Permutations[j] = newj
		}
		// Now apply  permutations list to outputMsgs
		newOutMsgs := make([]realtime.RealtimeSlot, BatchSize)
		for j := uint64(0); j < BatchSize; j++ {
			newOutMsgs[rounds[i].Permutations[j]] = outputMsgs[j]
		}
		outputMsgs = newOutMsgs
	}

	MultiNodeTest(nodeCount, BatchSize, &grp, rounds, inputMsgs, outputMsgs, t)
}
