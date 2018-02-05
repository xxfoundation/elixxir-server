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
	outStr := fmt.Sprintf("\tPrime: 101, Generator: %s, CypherPublicKey: %s, " +
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

	fmt.Printf("SHARE:\n")
	fmt.Printf("%v", RoundText(&grp, round))

	if shareResult.PartialRoundPublicCypherKey.Cmp(cyclic.NewInt(20)) != 0 {
		t.Errorf("SHARE failed, expected 20, got %s",
			shareResult.PartialRoundPublicCypherKey.Text(10))
	}

	// DECRYPT PHASE
	var decMsg services.Slot
	decMsg = &precomputation.SlotDecrypt{
		Slot:                         0,
		EncryptedMessageKeys:         cyclic.NewInt(1),
		PartialMessageCypherText:     cyclic.NewInt(1),
		EncryptedRecipientIDKeys:     cyclic.NewInt(1),
		PartialRecipientIDCypherText: cyclic.NewInt(1),
	}
	Decrypt := services.DispatchCryptop(&grp, precomputation.Decrypt{},
		nil, nil, round)

	// PERMUTE PHASE
	Permute := services.DispatchCryptop(&grp, precomputation.Permute{},
		nil, nil, round)

	go func(in, out chan *services.Slot) {
		iv := <-in
		is := precomputation.SlotPermute(*((*iv).(*precomputation.SlotDecrypt)))

		fmt.Printf("DECRYPT:\n  EncryptedMessageKeys: %s, " +
			"EncryptedRecipientIDKeys: %s,\n"+
			"  PartialMessageCypherText: %s, PartialRecipientIDCypherText: %s\n",
			is.EncryptedMessageKeys.Text(10), is.EncryptedRecipientIDKeys.Text(10),
			is.PartialMessageCypherText.Text(10),
			is.PartialRecipientIDCypherText.Text(10))

		expectedDecrypt := []*cyclic.Int{
			cyclic.NewInt(32), cyclic.NewInt(35),
			cyclic.NewInt(30), cyclic.NewInt(45),
		}
		if is.EncryptedMessageKeys.Cmp(expectedDecrypt[0]) != 0 {
			t.Errorf("DECRYPT failed EncryptedMessageKeys. Got: %s Expected: %s",
				is.EncryptedMessageKeys.Text(10), expectedDecrypt[0].Text(10))
		}
		if is.EncryptedRecipientIDKeys.Cmp(expectedDecrypt[1]) != 0 {
			t.Errorf("DECRYPT failed EncryptedRecipientIDKeys. Got: %s Expected: %s",
				is.EncryptedRecipientIDKeys.Text(10), expectedDecrypt[1].Text(10))
		}
		if is.PartialMessageCypherText.Cmp(expectedDecrypt[2]) != 0 {
			t.Errorf("DECRYPT failed PartialMessageCypherText. Got: %s Expected: %s",
				is.PartialMessageCypherText.Text(10), expectedDecrypt[2].Text(10))
		}
		if is.PartialRecipientIDCypherText.Cmp(expectedDecrypt[3]) != 0 {
			t.Errorf("DECRYPT failed PartialRecipientIDCypherText. Got: %s " +
				"Expected: %s", is.PartialRecipientIDCypherText.Text(10),
				expectedDecrypt[3].Text(10))
		}

		ov := services.Slot(&is)
		out <- &ov
	}(Decrypt.OutChannel, Permute.InChannel)

	// // ENCRYPT PHASE
	Encrypt := services.DispatchCryptop(&grp, precomputation.Encrypt{},
		nil, nil, round)

	go func(in, out chan *services.Slot) {
		iv := <-in
		pm := (*iv).(*precomputation.SlotPermute)
		se := precomputation.SlotEncrypt{
			Slot:                     pm.Slot,
			EncryptedMessageKeys:     pm.EncryptedMessageKeys,
			PartialMessageCypherText: pm.PartialMessageCypherText,
		}

		fmt.Printf("PERMUTE:\n  EncryptedMessageKeys: %s, " +
			"EncryptedRecipientIDKeys: %s,\n"+
			"  PartialMessageCypherText: %s, PartialRecipientIDCypherText: %s\n",
			pm.EncryptedMessageKeys.Text(10), pm.EncryptedRecipientIDKeys.Text(10),
			pm.PartialMessageCypherText.Text(10),
			pm.PartialRecipientIDCypherText.Text(10))

		expectedPermute := []*cyclic.Int{
			cyclic.NewInt(83), cyclic.NewInt(17),
			cyclic.NewInt(1), cyclic.NewInt(88),
		}
		if pm.EncryptedMessageKeys.Cmp(expectedPermute[0]) != 0 {
			t.Errorf("PERMUTE failed EncryptedMessageKeys. Got: %s Expected: %s",
				pm.EncryptedMessageKeys.Text(10), expectedPermute[0].Text(10))
		}
		if pm.EncryptedRecipientIDKeys.Cmp(expectedPermute[1]) != 0 {
			t.Errorf("PERMUTE failed EncryptedRecipientIDKeys. Got: %s Expected: %s",
				pm.EncryptedRecipientIDKeys.Text(10), expectedPermute[1].Text(10))
		}
		if pm.PartialMessageCypherText.Cmp(expectedPermute[2]) != 0 {
			t.Errorf("PERMUTE failed PartialMessageCypherText. Got: %s Expected: %s",
				pm.PartialMessageCypherText.Text(10), expectedPermute[2].Text(10))
		}
		if pm.PartialRecipientIDCypherText.Cmp(expectedPermute[3]) != 0 {
			t.Errorf("PERMUTE failed PartialRecipientIDCypherText. Got: %s " +
				"Expected: %s", pm.PartialRecipientIDCypherText.Text(10),
				expectedPermute[3].Text(10))
		}

		// Save the results to LastNode, which we don't have to check
		// because we are the only node
		i := pm.Slot
		round.LastNode.RecipientCypherText[i].Set(pm.PartialRecipientIDCypherText)
		round.LastNode.EncryptedRecipientPrecomputation[i].Set(
			pm.EncryptedRecipientIDKeys)

		ov := services.Slot(&se)
		out <- &ov
	}(Permute.OutChannel, Encrypt.InChannel)

	// REVEAL PHASE
	Reveal := services.DispatchCryptop(&grp, precomputation.Reveal{},
		nil, nil, round)

	go func(in, out chan *services.Slot) {
		iv := <-in
		pm := (*iv).(*precomputation.SlotEncrypt)
		i := pm.Slot
		se := precomputation.SlotReveal{
			Slot: i,
			PartialMessageCypherText:   pm.PartialMessageCypherText,
			PartialRecipientCypherText: round.LastNode.RecipientCypherText[i],
		}

		fmt.Printf("ENCRYPT:\n  EncryptedMessageKeys: %s, " +
			"PartialMessageCypherText: %s\n", pm.EncryptedMessageKeys.Text(10),
			pm.PartialMessageCypherText.Text(10))

		expectedEncrypt := []*cyclic.Int{
			cyclic.NewInt(57), cyclic.NewInt(9),
		}
		if pm.EncryptedMessageKeys.Cmp(expectedEncrypt[0]) != 0 {
			t.Errorf("ENCRYPT failed EncryptedMessageKeys. Got: %s Expected: %s",
				pm.EncryptedMessageKeys.Text(10), expectedEncrypt[0].Text(10))
		}
		if pm.PartialMessageCypherText.Cmp(expectedEncrypt[1]) != 0 {
			t.Errorf("ENCRYPT failed EncryptedRecipientIDKeys. Got: %s Expected: %s",
				pm.PartialMessageCypherText.Text(10), expectedEncrypt[1].Text(10))
		}

		// Save the results to LastNode
		round.LastNode.EncryptedMessagePrecomputation[i].Set(
			pm.EncryptedMessageKeys)
		ov := services.Slot(&se)
		out <- &ov
	}(Encrypt.OutChannel, Reveal.InChannel)

	// STRIP PHASE
	Strip := services.DispatchCryptop(&grp, precomputation.Strip{},
		nil, nil, round)

	go func(in, out chan *services.Slot) {
		iv := <-in
		pm := (*iv).(*precomputation.SlotReveal)
		i := pm.Slot
		se := precomputation.SlotStripIn{
			Slot: i,
			RoundMessagePrivateKey:   pm.PartialMessageCypherText,
			RoundRecipientPrivateKey: pm.PartialRecipientCypherText,
		}

		fmt.Printf("REVEAL:\n  RoundMessagePrivateKey: %s, " +
			"RoundRecipientPrivateKey: %s\n", se.RoundMessagePrivateKey.Text(10),
			se.RoundRecipientPrivateKey.Text(10))
		expectedReveal := []*cyclic.Int{
			cyclic.NewInt(20), cyclic.NewInt(68),
		}
		if se.RoundMessagePrivateKey.Cmp(expectedReveal[0]) != 0 {
			t.Errorf("REVEAL failed RoundMessagePrivateKey. Got: %s Expected: %s",
				se.RoundMessagePrivateKey.Text(10), expectedReveal[0].Text(10))
		}
		if se.RoundRecipientPrivateKey.Cmp(expectedReveal[1]) != 0 {
			t.Errorf("REVEAL failed RoundRecipientPrivateKey. Got: %s Expected: %s",
				se.RoundRecipientPrivateKey.Text(10), expectedReveal[1].Text(10))
		}

		ov := services.Slot(&se)
		out <- &ov
	}(Reveal.OutChannel, Strip.InChannel)

	// KICK OFF PRECOMPUTATION and save
	Decrypt.InChannel <- &decMsg
	rtn := <-Strip.OutChannel
	es := (*rtn).(*precomputation.SlotStripOut)

	round.LastNode.MessagePrecomputation[es.Slot] = es.MessagePrecomputation
	round.LastNode.RecipientPrecomputation[es.Slot] = es.RecipientPrecomputation

	fmt.Printf("STRIP:\n  MessagePrecomputation: %s, " +
		"RecipientPrecomputation: %s\n", es.MessagePrecomputation.Text(10),
		es.RecipientPrecomputation.Text(10))
	expectedStrip := []*cyclic.Int{
		cyclic.NewInt(18), cyclic.NewInt(76),
	}
	if es.MessagePrecomputation.Cmp(expectedStrip[0]) != 0 {
		t.Errorf("STRIP failed MessagePrecomputation. Got: %s Expected: %s",
			es.MessagePrecomputation.Text(10), expectedStrip[0].Text(10))
	}
	if es.RecipientPrecomputation.Cmp(expectedStrip[1]) != 0 {
		t.Errorf("STRIP failed RecipientPrecomputation. Got: %s Expected: %s",
			es.RecipientPrecomputation.Text(10), expectedStrip[1].Text(10))
	}


	MP, RP := ComputeSingleNodePrecomputation(&grp, round)

	if MP.Cmp(es.MessagePrecomputation) != 0 {
		t.Errorf("Message Precomputation Incorrect! Expected: %s, Received: %s\n",
			MP.Text(10), es.MessagePrecomputation.Text(10))
	}

	if RP.Cmp(es.RecipientPrecomputation) != 0 {
		t.Errorf("Recipient Precomputation Incorrect! Expected: %s, Received: %s\n",
			RP.Text(10), es.RecipientPrecomputation.Text(10))
	}

	// ----- REALTIME ----- //
	inputMsg := services.Slot(&realtime.SlotDecryptIn{
		Slot:                 0,
		SenderID:             1,
		EncryptedMessage:     cyclic.NewInt(31),
		EncryptedRecipientID: cyclic.NewInt(1),
		TransmissionKey:      cyclic.NewInt(1),
	})

	// DECRYPT PHASE
	RTDecrypt := services.DispatchCryptop(&grp, realtime.Decrypt{},
		nil, nil, round)

	// PERMUTE PHASE
	RTPermute := services.DispatchCryptop(&grp, realtime.Permute{},
		nil, nil, round)

	go func(in, out chan *services.Slot) {
		iv := <-in
		is := (*iv).(*realtime.SlotDecryptOut)
		ov := services.Slot(&realtime.SlotPermute{
			Slot:                 is.Slot,
			EncryptedMessage:     is.EncryptedMessage,
			EncryptedRecipientID: is.EncryptedRecipientID,
		})

		fmt.Printf("RTDECRYPT:\n  EncryptedMessage: %s, EncryptedRecipientID: %s\n",
			is.EncryptedMessage.Text(10),
			is.EncryptedRecipientID.Text(10))
		expectedRTDecrypt := []*cyclic.Int{
			cyclic.NewInt(75), cyclic.NewInt(72),
		}
		if is.EncryptedMessage.Cmp(expectedRTDecrypt[0]) != 0 {
			t.Errorf("RTDECRYPT failed EncryptedMessage. Got: %s Expected: %s",
				is.EncryptedMessage.Text(10), expectedRTDecrypt[0].Text(10))
		}
		if is.EncryptedRecipientID.Cmp(expectedRTDecrypt[1]) != 0 {
			t.Errorf("RTDECRYPT failed EncryptedRecipientID. Got: %s Expected: %s",
				is.EncryptedRecipientID.Text(10), expectedRTDecrypt[1].Text(10))
		}

		out <- &ov
	}(RTDecrypt.OutChannel, RTPermute.InChannel)

	// IDENTIFY PHASE
	RTIdentify := services.DispatchCryptop(&grp, realtime.Identify{},
		nil, nil, round)

	RTDecrypt.InChannel <- &inputMsg
	rtnPrm := <-RTPermute.OutChannel
	esPrm := (*rtnPrm).(*realtime.SlotPermute)
	ovPrm := services.Slot(&realtime.SlotIdentify{
		Slot:                 esPrm.Slot,
		EncryptedRecipientID: esPrm.EncryptedRecipientID,
	})
	TmpMsg := esPrm.EncryptedMessage
	fmt.Printf("RTPERMUTE:\n  EncryptedRecipientID: %s\n",
		esPrm.EncryptedRecipientID.Text(10))
	expectedRTPermute := []*cyclic.Int{
		cyclic.NewInt(4),
	}
	if esPrm.EncryptedRecipientID.Cmp(expectedRTPermute[0]) != 0 {
		t.Errorf("RTPERMUTE failed EncryptedRecipientID. Got: %s Expected: %s",
			esPrm.EncryptedRecipientID.Text(10), expectedRTPermute[0].Text(10))
	}

	RTIdentify.InChannel <- &ovPrm
	rtnTmp := <-RTIdentify.OutChannel
	esTmp := (*rtnTmp).(*realtime.SlotIdentify)
	rID, _ := strconv.ParseUint(esTmp.EncryptedRecipientID.Text(10), 10, 64)
	inputMsgPostID := services.Slot(&realtime.SlotEncryptIn{
		Slot:             esTmp.Slot,
		RecipientID:      rID,
		EncryptedMessage: TmpMsg,
		ReceptionKey:     cyclic.NewInt(1),
	})
	fmt.Printf("RTIDENTIFY:\n  RecipientID: %s\n",
		esTmp.EncryptedRecipientID.Text(10))
	expectedRTIdentify := []*cyclic.Int{
		cyclic.NewInt(1),
	}
	if esTmp.EncryptedRecipientID.Cmp(expectedRTIdentify[0]) != 0 {
		t.Errorf("RTIDENTIFY failed EncryptedRecipientID. Got: %s Expected: %s",
			esTmp.EncryptedRecipientID.Text(10), expectedRTIdentify[0].Text(10))
	}

	// ENCRYPT PHASE
	RTEncrypt := services.DispatchCryptop(&grp, realtime.Encrypt{},
		nil, nil, round)

	// PEEL PHASE
	RTPeel := services.DispatchCryptop(&grp, realtime.Peel{},
		nil, nil, round)

	go func(in, out chan *services.Slot) {
		iv := <-in
		is := realtime.SlotPeel(*((*iv).(*realtime.SlotEncryptOut)))
		ov := services.Slot(&is)

		fmt.Printf("RTENCRYPT:\n  EncryptedMessage: %s\n",
			is.EncryptedMessage.Text(10))
		expectedRTEncrypt := []*cyclic.Int{
			cyclic.NewInt(41),
		}
		if is.EncryptedMessage.Cmp(expectedRTEncrypt[0]) != 0 {
			t.Errorf("RTENCRYPT failed EncryptedMessage. Got: %s Expected: %s",
				is.EncryptedMessage.Text(10), expectedRTEncrypt[0].Text(10))
		}

		out <- &ov
	}(RTEncrypt.OutChannel, RTPeel.InChannel)

	// KICK OFF RT COMPUTATION
	RTEncrypt.InChannel <- &inputMsgPostID
	rtnRT := <-RTPeel.OutChannel
	esRT := (*rtnRT).(*realtime.SlotPeel)

	fmt.Printf("RTPEEL:\n  EncryptedMessage: %s\n",
		esRT.EncryptedMessage.Text(10))
	expectedRTPeel := []*cyclic.Int{
		cyclic.NewInt(31),
	}
	if esRT.EncryptedMessage.Cmp(expectedRTPeel[0]) != 0 {
		t.Errorf("RTPEEL failed EncryptedMessage. Got: %s Expected: %s",
			esRT.EncryptedMessage.Text(10), expectedRTPeel[0].Text(10))
	}

	fmt.Println("Final Results: Slot: %d, Recipient ID: %d, Message: %s\n",
		esRT.Slot, esRT.RecipientID,
		esRT.EncryptedMessage.Text(10))
}


// Convert Decrypt output slot to Permute input slot
func DecryptPermuteTranslate(decrypt, permute chan *services.Slot) {
	for decryptSlot := range decrypt {
		is := precomputation.SlotPermute(*((*decryptSlot).(
			*precomputation.SlotDecrypt)))
		sp := services.Slot(&is)
		permute <- &sp
	}
}

// Convert Permute output slot to Encrypt input slot
func PermuteEncryptTranslate(permute, encrypt chan *services.Slot,
	round *globals.Round) {
	for permuteSlot := range permute {
		is := (*permuteSlot).(*precomputation.SlotPermute)
		se := services.Slot(&precomputation.SlotEncrypt{
			Slot:                     is.Slot,
			EncryptedMessageKeys:     is.EncryptedMessageKeys,
			PartialMessageCypherText: is.PartialMessageCypherText,
		})
		// Save LastNode Data to Round
		i := is.Slot
		round.LastNode.RecipientCypherText[i].Set(is.PartialRecipientIDCypherText)
		round.LastNode.EncryptedRecipientPrecomputation[i].Set(
			is.EncryptedRecipientIDKeys)
		encrypt <- &se
	}
}

// Convert Encrypt output slot to Reveal input slot
func EncryptRevealTranslate(encrypt, reveal chan *services.Slot,
	round *globals.Round) {
	for encryptSlot := range encrypt {
		is := (*encryptSlot).(*precomputation.SlotEncrypt)
		i := is.Slot
		sr := services.Slot(&precomputation.SlotReveal{
			Slot: i,
			PartialMessageCypherText:   is.PartialMessageCypherText,
			PartialRecipientCypherText: round.LastNode.RecipientCypherText[i],
		})
		round.LastNode.EncryptedMessagePrecomputation[i].Set(
			is.EncryptedMessageKeys)
		reveal <- &sr
	}
}

// Convert Reveal output slot to Strip input slot
func RevealStripTranslate(reveal, strip chan *services.Slot) {
	for revealSlot := range reveal {
		is := (*revealSlot).(*precomputation.SlotReveal)
		i := is.Slot
		ss := services.Slot(&precomputation.SlotStripIn{
			Slot: i,
			RoundMessagePrivateKey:   is.PartialMessageCypherText,
			RoundRecipientPrivateKey: is.PartialRecipientCypherText,
		})
		strip <- &ss
	}
}

// Convert RTDecrypt output slot to RTPermute input slot
func RTDecryptRTPermuteTranslate(decrypt, permute chan *services.Slot) {
	for decryptSlot := range decrypt {
		is := (*decryptSlot).(*realtime.SlotDecryptOut)
		ov := services.Slot(&realtime.SlotPermute{
			Slot:                 is.Slot,
			EncryptedMessage:     is.EncryptedMessage,
			EncryptedRecipientID: is.EncryptedRecipientID,
		})
		permute <- &ov
	}
}

func RTPermuteRTIdentifyTranslate(permute, identify chan *services.Slot,
	outMsgs []*cyclic.Int) {
	for permuteSlot := range permute {
		esPrm := (*permuteSlot).(*realtime.SlotPermute)
		ovPrm := services.Slot(&realtime.SlotIdentify{
			Slot:                 esPrm.Slot,
			EncryptedRecipientID: esPrm.EncryptedRecipientID,
		})
		fmt.Printf("SLOT: %d", esPrm.Slot)
		outMsgs[esPrm.Slot].Set(esPrm.EncryptedMessage)
		identify <- &ovPrm
	}
}

func RTIdentifyRTEncryptTranslate(identify, encrypt chan *services.Slot,
	inMsgs[]*cyclic.Int) {
	for identifySlot := range identify {
		esTmp := (*identifySlot).(*realtime.SlotIdentify)
		rID, _ := strconv.ParseUint(esTmp.EncryptedRecipientID.Text(10), 10, 64)
		inputMsgPostID := services.Slot(&realtime.SlotEncryptIn{
			Slot:             esTmp.Slot,
			RecipientID:      rID,
			EncryptedMessage: inMsgs[esTmp.Slot],
			ReceptionKey:     cyclic.NewInt(1),
		})
		encrypt <- &inputMsgPostID
	}
}

func RTEncryptRTPeelTranslate(encrypt, peel chan *services.Slot) {
	for encryptSlot := range encrypt {
		is := realtime.SlotPeel(*((*encryptSlot).(*realtime.SlotEncryptOut)))
		ov := services.Slot(&is)
		peel <- &ov
	}
}

func RTDecryptRTDecryptTranslate(in, out chan *services.Slot) {
	for is := range in {
		o := (*is).(*realtime.SlotDecryptOut)
		os := services.Slot(&realtime.SlotDecryptIn{
			Slot: o.Slot,
			SenderID: o.SenderID,
			EncryptedMessage: o.EncryptedMessage,
			EncryptedRecipientID: o.EncryptedRecipientID,
			TransmissionKey: cyclic.NewInt(1), // WTF? FIXME
		})
		out <- &os
	}
}

func RTEncryptRTEncryptTranslate(in, out chan *services.Slot) {
	for is := range in {
		o := (*is).(*realtime.SlotEncryptOut)
		os := services.Slot(&realtime.SlotEncryptIn{
			Slot: o.Slot,
			RecipientID: o.RecipientID,
			EncryptedMessage: o.EncryptedMessage,
			ReceptionKey: cyclic.NewInt(1), // FIXME
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
		nil, nil, Node1Round)
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
	go DecryptPermuteTranslate(N2Decrypt.OutChannel, N1Permute.InChannel)

	// Run Generate
	genMsg := services.Slot(&precomputation.SlotGeneration{Slot: 0})
	N1Generation.InChannel <- &genMsg
	_ = <-N1Generation.OutChannel
	N2Generation.InChannel <- &genMsg
	_ = <-N2Generation.OutChannel

	fmt.Printf("2 NODE GENERATION RESULTS: \n")
	fmt.Printf("%v", RoundText(&grp, Node1Round))
	fmt.Printf("%v", RoundText(&grp, Node2Round))

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

	fmt.Printf("2 NODE SHARE RESULTS: \n")
	fmt.Printf("%v", RoundText(&grp, Node2Round))
	fmt.Printf("%v", RoundText(&grp, Node1Round))

	// Now finish precomputation
	decMsg := services.Slot(&precomputation.SlotDecrypt{
		Slot:                         0,
		EncryptedMessageKeys:         cyclic.NewInt(1),
		PartialMessageCypherText:     cyclic.NewInt(1),
		EncryptedRecipientIDKeys:     cyclic.NewInt(1),
		PartialRecipientIDCypherText: cyclic.NewInt(1),
	})
	N1Decrypt.InChannel <- &decMsg
	rtn := <-N2Strip.OutChannel
	es := (*rtn).(*precomputation.SlotStripOut)

	Node2Round.LastNode.MessagePrecomputation[es.Slot] = es.MessagePrecomputation
	Node2Round.LastNode.RecipientPrecomputation[es.Slot] =
		es.RecipientPrecomputation
	fmt.Printf("2 NODE STRIP:\n  MessagePrecomputation: %s, " +
		"RecipientPrecomputation: %s\n", es.MessagePrecomputation.Text(10),
		es.RecipientPrecomputation.Text(10))

	// Check precomputation
	MP, RP := ComputePrecomputation(&grp,
		[]*globals.Round{Node1Round, Node2Round})

	if MP.Cmp(es.MessagePrecomputation) != 0 {
		t.Errorf("Message Precomputation Incorrect! Expected: %s, Received: %s\n",
			MP.Text(10), es.MessagePrecomputation.Text(10))
	}
	if RP.Cmp(es.RecipientPrecomputation) != 0 {
		t.Errorf("Recipient Precomputation Incorrect! Expected: %s, Received: %s\n",
			RP.Text(10), es.RecipientPrecomputation.Text(10))
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

	inputMsg := services.Slot(&realtime.SlotDecryptIn{
		Slot:                 0,
		SenderID:             1,
		EncryptedMessage:     cyclic.NewInt(42), // Meaning of Life
		EncryptedRecipientID: cyclic.NewInt(1),
		TransmissionKey:      cyclic.NewInt(1),
	})
	N1RTDecrypt.InChannel <- &inputMsg
	rtnRT := <-N2RTPeel.OutChannel
	esRT := (*rtnRT).(*realtime.SlotPeel)
	fmt.Printf("RTPEEL:\n  EncryptedMessage: %s\n",
		esRT.EncryptedMessage.Text(10))
	expectedRTPeel := []*cyclic.Int{
		cyclic.NewInt(42),
	}
	if esRT.EncryptedMessage.Cmp(expectedRTPeel[0]) != 0 {
		t.Errorf("RTPEEL failed EncryptedMessage. Got: %s Expected: %s",
			esRT.EncryptedMessage.Text(10), expectedRTPeel[0].Text(10))
	}

	fmt.Println("Final Results: Slot: %d, Recipient ID: %d, Message: %s\n",
		esRT.Slot, esRT.RecipientID,
		esRT.EncryptedMessage.Text(10))
}


// Perform an end to end test of the precomputation with batchsize 1,
// then use it to send the message through a 2-node system to smoke test
// the cryptographic operations.
func MultiNodeTest(nodeCount int, BatchSize uint64,
	group *cyclic.Group, keys []*globals.Round,
	inputMsgs []realtime.SlotDecryptIn, t *testing.T) {

	// Init Round Vars
	var rounds []*globals.Round
	var LastRound *globals.Round
	for i := 0; i < nodeCount; i++ {
		rounds = append(rounds, globals.NewRound(BatchSize))
		rounds[i].CypherPublicKey = cyclic.NewInt(0)
		// Last Node initialization
		if i == (nodeCount - 1) {
			globals.InitLastNode(rounds[i])
			LastRound = rounds[i]
		}
	}

	// ----- PRECOMPUTATION ----- //
	generations := make([]*services.ThreadController, nodeCount)
	shares := make([]*services.ThreadController, nodeCount)
	decrypts := make([]*services.ThreadController, nodeCount)
	permutes := make([]*services.ThreadController, nodeCount)
	encrypts := make([]*services.ThreadController, nodeCount)
	reveals := make([]*services.ThreadController, nodeCount)
	for i := 0; i < nodeCount; i++ {
		generations[i] = services.DispatchCryptop(group,
			precomputation.Generation{}, nil, nil, rounds[i])

		if i == 0 {
			shares[i] = services.DispatchCryptop(group, precomputation.Share{},
				nil, nil, rounds[i])
			decrypts[i] = services.DispatchCryptop(group, precomputation.Decrypt{},
				nil, nil, rounds[i])
			permutes[i] = services.DispatchCryptop(group, precomputation.Permute{},
				nil, nil, rounds[i])
			encrypts[i] = services.DispatchCryptop(group, precomputation.Encrypt{},
				nil, nil, rounds[i])
			reveals[i] = services.DispatchCryptop(group, precomputation.Reveal{},
				nil, nil, rounds[i])
		} else {
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
	go DecryptPermuteTranslate(decrypts[nodeCount-1].OutChannel,
		permutes[0].InChannel)

	// Run Generate
	genMsg := services.Slot(&precomputation.SlotGeneration{Slot: 0})
	for i := 0; i < nodeCount; i++ {
		generations[i].InChannel <- &genMsg
		_ = <-generations[i].OutChannel
	}

	fmt.Printf("%d NODE GENERATION RESULTS: \n", nodeCount)
	for i := 0; i < nodeCount; i++ {
		fmt.Printf("%v", RoundText(group, rounds[i]))
	}

	// TODO: Pre-can the keys to use here if necessary.

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

	fmt.Printf("%d NODE SHARE RESULTS: \n", nodeCount)
	for i := 0; i < nodeCount; i++ {
		fmt.Printf("%v", RoundText(group, rounds[i]))
	}

	// Now finish precomputation
	decMsg := services.Slot(&precomputation.SlotDecrypt{
		Slot:                         0,
		EncryptedMessageKeys:         cyclic.NewInt(1),
		PartialMessageCypherText:     cyclic.NewInt(1),
		EncryptedRecipientIDKeys:     cyclic.NewInt(1),
		PartialRecipientIDCypherText: cyclic.NewInt(1),
	})
	decrypts[0].InChannel <- &decMsg
	rtn := <-LNStrip.OutChannel
	es := (*rtn).(*precomputation.SlotStripOut)

	LastRound.LastNode.MessagePrecomputation[es.Slot] = es.MessagePrecomputation
	LastRound.LastNode.RecipientPrecomputation[es.Slot] =
		es.RecipientPrecomputation

	fmt.Printf("%d NODE STRIP:\n  MessagePrecomputation: %s, " +
		"RecipientPrecomputation: %s\n", nodeCount,
		es.MessagePrecomputation.Text(10),
		es.RecipientPrecomputation.Text(10))

	// Check precomputation
	MP, RP := ComputePrecomputation(group, rounds)

	if MP.Cmp(es.MessagePrecomputation) != 0 {
		t.Errorf("Message Precomputation Incorrect! Expected: %s, Received: %s\n",
			MP.Text(10), es.MessagePrecomputation.Text(10))
	}
	if RP.Cmp(es.RecipientPrecomputation) != 0 {
		t.Errorf("Recipient Precomputation Incorrect! Expected: %s, Received: %s\n",
			RP.Text(10), es.RecipientPrecomputation.Text(10))
	}

	// ----- REALTIME ----- //
	IntermediateMsgs := make([]*cyclic.Int, nodeCount)
	rtdecrypts := make([]*services.ThreadController, nodeCount)
	rtpermutes := make([]*services.ThreadController, nodeCount)
	rtencrypts := make([]*services.ThreadController, nodeCount)
	for i := 0; i < nodeCount; i++ {
		IntermediateMsgs[i] = cyclic.NewInt(0)

		rtdecrypts[i] = services.DispatchCryptop(group,
			realtime.Decrypt{}, nil, nil, rounds[i])
		if i == 0 {
			rtpermutes[i] = services.DispatchCryptop(group,
				realtime.Permute{}, nil, nil, rounds[i])
		} else {
			rtpermutes[i] = services.DispatchCryptop(group,
				realtime.Permute{}, rtpermutes[i-1].OutChannel, nil, rounds[i])
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
	go RTPermuteRTIdentifyTranslate(rtpermutes[nodeCount-1].OutChannel,
		LNRTIdentify.InChannel, IntermediateMsgs)
	go RTIdentifyRTEncryptTranslate(LNRTIdentify.OutChannel,
		rtencrypts[0].InChannel, IntermediateMsgs)
	go RTEncryptRTPeelTranslate(rtencrypts[nodeCount-1].OutChannel,
		LNRTPeel.InChannel)

	expectedRTPeel := make([]*cyclic.Int, BatchSize)
	for i := uint64(0); i < BatchSize; i++ {
		expectedRTPeel[i] = cyclic.NewInt(0)
		expectedRTPeel[i].Set(inputMsgs[i].EncryptedMessage)
		in := services.Slot(&inputMsgs[i])
		rtdecrypts[0].InChannel <- &in
	}

	for i := uint64(0); i < BatchSize; i++ {
		rtnRT := <-LNRTPeel.OutChannel
		esRT := (*rtnRT).(*realtime.SlotPeel)
		fmt.Printf("RTPEEL:\n  EncryptedMessage: %s\n",
			esRT.EncryptedMessage.Text(10))

		if esRT.EncryptedMessage.Cmp(expectedRTPeel[i]) != 0 {
			t.Errorf("RTPEEL failed EncryptedMessage. Got: %s Expected: %s",
				esRT.EncryptedMessage.Text(10), expectedRTPeel[0].Text(10))
		}

		fmt.Println("Final Results: Slot: %d, Recipient ID: %d, Message: %s\n",
			esRT.Slot, esRT.RecipientID,
			esRT.EncryptedMessage.Text(10))
	}
}

func Test3NodeE2E(t *testing.T) {
	nodeCount := 3
	BatchSize := uint64(1)
	rng := cyclic.NewRandom(cyclic.NewInt(0), cyclic.NewInt(1000))
	grp := cyclic.NewGroup(cyclic.NewInt(101), cyclic.NewInt(5), cyclic.NewInt(4),
		rng)
	inputMsgs := make([]realtime.SlotDecryptIn, BatchSize)
	for i := uint64(0); i < BatchSize; i++ {
		inputMsgs[i] = realtime.SlotDecryptIn{
			Slot:                 i,
			SenderID:             i+1,
			EncryptedMessage:     cyclic.NewInt(42 + int64(i)), // Meaning of Life
			EncryptedRecipientID: cyclic.NewInt(1),
			TransmissionKey:      cyclic.NewInt(1),
		}
	}
	MultiNodeTest(nodeCount, BatchSize, &grp, nil, inputMsgs, t)
}

func Test1NodePermuteE2E(t *testing.T) {
	nodeCount := 1
	BatchSize := uint64(1)
	rng := cyclic.NewRandom(cyclic.NewInt(0), cyclic.NewInt(1000))
	grp := cyclic.NewGroup(cyclic.NewInt(101), cyclic.NewInt(5), cyclic.NewInt(4),
		rng)
	inputMsgs := make([]realtime.SlotDecryptIn, BatchSize)
	for i := uint64(0); i < BatchSize; i++ {
		inputMsgs[i] = realtime.SlotDecryptIn{
			Slot:                 i,
			SenderID:             i+1,
			EncryptedMessage:     cyclic.NewInt(42 + int64(i)), // Meaning of Life
			EncryptedRecipientID: cyclic.NewInt(1),
			TransmissionKey:      cyclic.NewInt(1),
		}
	}
	MultiNodeTest(nodeCount, BatchSize, &grp, nil, inputMsgs, t)
}
