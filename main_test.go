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
	outStr := fmt.Sprintf("\nPrime: 101, Generator: %s, CypherPublicKey: %s," +
		"Z: %s\n", g.G.Text(10), n.CypherPublicKey.Text(10), n.Z.Text(10))
	outStr += fmt.Sprintf("Permutations: %v\n", n.Permutations)
	rfmt := "Round[%d]: \t R(%s, %s, %s) S(%s, %s, %s) T(%s, %s, %s) \n" +
		"\t\t\t\t\t\t U(%s, %s, %s) V(%s, %s, %s) \n"
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

	RootingTest(&grp)
	RootingTestDouble(&grp)
	RootingTestTriple(&grp)

	//fmt.Printf("%v", RoundText(&grp, round))

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

	// TODO: This phase requires us to use pre-cooked crypto values. We run
	// the step here then overwrite the values that were stored in the
	// round structure so we still get the same results. We should perform
	// the override here.

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

	t.Errorf("Got: %v", round.CypherPublicKey.Text(10))

	fmt.Printf("%v", RoundText(&grp, round))

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

	t.Errorf("What? %+v", rtn)

	MP, RP := ComputeSingleNodePrecomputation(&grp, round)

	if MP.Cmp(es.MessagePrecomputation) != 0 {
		t.Errorf("Message Precomputation Incorrect! Expected: %s, Received: %s\n",
			MP.Text(10), es.MessagePrecomputation.Text(10))
	}

	if RP.Cmp(es.RecipientPrecomputation) != 0 {
		t.Errorf("Message Precomputation Incorrect! Expected: %s, Received: %s\n",
			RP.Text(10), es.RecipientPrecomputation.Text(10))
	}

	// ----- REALTIME ----- //
	inputMsg := services.Slot(&realtime.SlotDecryptIn{
		Slot:                 0,
		SenderID:             1,
		EncryptedMessage:     cyclic.NewInt(3),
		EncryptedRecipientID: cyclic.NewInt(3),
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

		fmt.Printf("DECRYPT:\n  EncryptedMessage: %s, EncryptedRecipientID: %s\n",
			is.EncryptedMessage.Text(10),
			is.EncryptedRecipientID.Text(10))

		out <- &ov
	}(RTDecrypt.OutChannel, RTPermute.InChannel)

	// IDENTIFY PHASE
	RTIdentify := services.DispatchCryptop(&grp, realtime.Identify{},
		nil, nil, round)

	// FIXME
	RTDecrypt.InChannel <- &inputMsg
	rtnPrm := <-RTPermute.OutChannel
	esPrm := (*rtnPrm).(*realtime.SlotPermute)
	ovPrm := services.Slot(&realtime.SlotIdentify{
		Slot:                 esPrm.Slot,
		EncryptedRecipientID: esPrm.EncryptedRecipientID,
	})
	TmpMsg := esPrm.EncryptedMessage

	// HACK HACK HACK FIXME FIXME
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
		out <- &ov
	}(RTEncrypt.OutChannel, RTPeel.InChannel)

	// KICK OFF RT COMPUTATION
	RTEncrypt.InChannel <- &inputMsgPostID
	rtnRT := <-RTPeel.OutChannel
	esRT := (*rtnRT).(*realtime.SlotPeel)

	fmt.Println("Final Results: %d, %d, %s",
		esRT.Slot, esRT.RecipientID,
		esRT.EncryptedMessage.Text(10))

}
