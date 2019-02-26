////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package main

import (
	jww "github.com/spf13/jwalterweatherman"

	"fmt"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/benchmark"
	"gitlab.com/elixxir/server/cryptops/precomputation"
	"gitlab.com/elixxir/server/cryptops/realtime"
	"gitlab.com/elixxir/server/globals"
	"gitlab.com/elixxir/server/services"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	// Set log level high for main testing to disable MIC errors, etc
	jww.SetStdoutThreshold(jww.LevelFatal)
	os.Exit(m.Run())
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
	t.Logf("%v", benchmark.RoundText(&grp, round))

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
		AssociatedDataCypher:         cyclic.NewInt(1),
		AssociatedDataPrecomputation: cyclic.NewInt(1),
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
			"AssociatedDataCypher: %s,\n"+
			"  MessagePrecomputation: %s, AssociatedDataPrecomputation: %s\n",
			is.MessageCypher.Text(10), is.AssociatedDataCypher.Text(10),
			is.MessagePrecomputation.Text(10),
			is.AssociatedDataPrecomputation.Text(10))

		expectedDecrypt := []*cyclic.Int{
			cyclic.NewInt(32), cyclic.NewInt(35),
			cyclic.NewInt(30), cyclic.NewInt(45),
		}
		if is.MessageCypher.Cmp(expectedDecrypt[0]) != 0 {
			t.Errorf("DECRYPT failed MessageCypher. Got: %s Expected: %s",
				is.MessageCypher.Text(10), expectedDecrypt[0].Text(10))
		}
		if is.AssociatedDataCypher.Cmp(expectedDecrypt[1]) != 0 {
			t.Errorf("DECRYPT failed AssociatedDataCypher. Got: %s Expected: %s",
				is.AssociatedDataCypher.Text(10), expectedDecrypt[1].Text(10))
		}
		if is.MessagePrecomputation.Cmp(expectedDecrypt[2]) != 0 {
			t.Errorf("DECRYPT failed MessagePrecomputation. Got: %s Expected: %s",
				is.MessagePrecomputation.Text(10), expectedDecrypt[2].Text(10))
		}
		if is.AssociatedDataPrecomputation.Cmp(expectedDecrypt[3]) != 0 {
			t.Errorf("DECRYPT failed AssociatedDataPrecomputation. Got: %s "+
				"Expected: %s", is.AssociatedDataPrecomputation.Text(10),
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
			"AssociatedDataCypher: %s,\n"+
			"  MessagePrecomputation: %s, AssociatedDataPrecomputation: %s\n",
			pm.MessageCypher.Text(10), pm.AssociatedDataCypher.Text(10),
			pm.MessagePrecomputation.Text(10),
			pm.AssociatedDataPrecomputation.Text(10))

		expectedPermute := []*cyclic.Int{
			cyclic.NewInt(83), cyclic.NewInt(17),
			cyclic.NewInt(1), cyclic.NewInt(88),
		}
		if pm.MessageCypher.Cmp(expectedPermute[0]) != 0 {
			t.Errorf("PERMUTE failed MessageCypher. Got: %s Expected: %s",
				pm.MessageCypher.Text(10), expectedPermute[0].Text(10))
		}
		if pm.AssociatedDataCypher.Cmp(expectedPermute[1]) != 0 {
			t.Errorf("PERMUTE failed AssociatedDataCypher. Got: %s Expected: %s",
				pm.AssociatedDataCypher.Text(10), expectedPermute[1].Text(10))
		}
		if pm.MessagePrecomputation.Cmp(expectedPermute[2]) != 0 {
			t.Errorf("PERMUTE failed MessagePrecomputation. Got: %s Expected: %s",
				pm.MessagePrecomputation.Text(10), expectedPermute[2].Text(10))
		}
		if pm.AssociatedDataPrecomputation.Cmp(expectedPermute[3]) != 0 {
			t.Errorf("PERMUTE failed AssociatedDataPrecomputation. Got: %s "+
				"Expected: %s", pm.AssociatedDataPrecomputation.Text(10),
				expectedPermute[3].Text(10))
		}

		// Save the results to LastNode, which we don't have to check
		// because we are the only node
		i := pm.Slot
		round.LastNode.RecipientCypherText[i].Set(pm.AssociatedDataPrecomputation)
		round.LastNode.EncryptedRecipientPrecomputation[i].Set(
			pm.AssociatedDataCypher)

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
			t.Errorf("ENCRYPT failed AssociatedDataCypher. Got: %s Expected: %s",
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
			"RoundRecipientPrivateKey: %s\n", pm.AssociatedDataPrecomputation.Text(10),
			pm.AssociatedDataPrecomputation.Text(10))
		expectedReveal := []*cyclic.Int{
			cyclic.NewInt(20), cyclic.NewInt(68),
		}
		if pm.MessagePrecomputation.Cmp(expectedReveal[0]) != 0 {
			t.Errorf("REVEAL failed RoundMessagePrivateKey. Got: %s Expected: %s",
				pm.MessagePrecomputation.Text(10), expectedReveal[0].Text(10))
		}
		if pm.AssociatedDataPrecomputation.Cmp(expectedReveal[1]) != 0 {
			t.Errorf("REVEAL failed RoundRecipientPrivateKey. Got: %s Expected: %s",
				pm.AssociatedDataPrecomputation.Text(10), expectedReveal[1].Text(10))
		}

		ov := services.Slot(pm)
		out <- &ov
	}(Reveal.OutChannel, Strip.InChannel)

	// KICK OFF PRECOMPUTATION and save
	Decrypt.InChannel <- &decMsg
	rtn := <-Strip.OutChannel
	es := (*rtn).(*precomputation.PrecomputationSlot)

	round.LastNode.MessagePrecomputation[es.Slot] = es.MessagePrecomputation
	round.LastNode.RecipientPrecomputation[es.Slot] = es.AssociatedDataPrecomputation

	t.Logf("STRIP:\n  MessagePrecomputation: %s, "+
		"RecipientPrecomputation: %s\n", es.MessagePrecomputation.Text(10),
		es.AssociatedDataPrecomputation.Text(10))
	expectedStrip := []*cyclic.Int{
		cyclic.NewInt(18), cyclic.NewInt(76),
	}
	if es.MessagePrecomputation.Cmp(expectedStrip[0]) != 0 {
		t.Errorf("STRIP failed MessagePrecomputation. Got: %s Expected: %s",
			es.MessagePrecomputation.Text(10), expectedStrip[0].Text(10))
	}
	if es.AssociatedDataPrecomputation.Cmp(expectedStrip[1]) != 0 {
		t.Errorf("STRIP failed RecipientPrecomputation. Got: %s Expected: %s",
			es.AssociatedDataPrecomputation.Text(10), expectedStrip[1].Text(10))
	}

	MP, RP := ComputeSingleNodePrecomputation(&grp, round)

	if MP.Cmp(es.MessagePrecomputation) != 0 {
		t.Errorf("Message Precomputation Incorrect! Expected: %s, Received: %s\n",
			MP.Text(10), es.MessagePrecomputation.Text(10))
	}

	if RP.Cmp(es.AssociatedDataPrecomputation) != 0 {
		t.Errorf("Recipient Precomputation Incorrect! Expected: %s, Received: %s\n",
			RP.Text(10), es.AssociatedDataPrecomputation.Text(10))
	}

	// ----- REALTIME ----- //
	inputMsg := services.Slot(&realtime.Slot{
		Slot:               0,
		CurrentID:          id.NewUserFromUint(1, t),
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
		is := (*iv).(*realtime.Slot)
		ov := services.Slot(&realtime.Slot{
			Slot:               is.Slot,
			Message:            is.Message,
			EncryptedRecipient: is.EncryptedRecipient,
		})

		t.Logf("RTDECRYPT:\n  EncryptedMessage: %s, EncryptedAssociatedData: %s\n",
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
			t.Errorf("RTDECRYPT failed EncryptedAssociatedData. Got: %s Expected: %s",
				is.EncryptedRecipient.Text(10), expectedRTDecrypt[1].Text(10))
		}

		out <- &ov
	}(RTDecrypt.OutChannel, RTPermute.InChannel)

	// IDENTIFY PHASE
	RTIdentify := services.DispatchCryptop(&grp, realtime.Identify{},
		nil, nil, round)

	RTDecrypt.InChannel <- &inputMsg
	rtnPrm := <-RTPermute.OutChannel
	esPrm := (*rtnPrm).(*realtime.Slot)
	ovPrm := services.Slot(&realtime.Slot{
		Slot:               esPrm.Slot,
		EncryptedRecipient: esPrm.EncryptedRecipient,
	})
	TmpMsg := esPrm.Message
	t.Logf("RTPERMUTE:\n  EncryptedAssociatedData: %s\n",
		esPrm.EncryptedRecipient.Text(10))
	expectedRTPermute := []*cyclic.Int{
		cyclic.NewInt(4),
	}
	if esPrm.EncryptedRecipient.Cmp(expectedRTPermute[0]) != 0 {
		t.Errorf("RTPERMUTE failed EncryptedAssociatedData. Got: %s Expected: %s",
			esPrm.EncryptedRecipient.Text(10), expectedRTPermute[0].Text(10))
	}

	RTIdentify.InChannel <- &ovPrm
	rtnTmp := <-RTIdentify.OutChannel
	esTmp := (*rtnTmp).(*realtime.Slot)
	rID := new(id.User).SetBytes(esTmp.EncryptedRecipient.
		LeftpadBytes(id.UserLen))
	copy(rID[:], esTmp.EncryptedRecipient.LeftpadBytes(id.UserLen))
	inputMsgPostID := services.Slot(&realtime.Slot{
		Slot:       esTmp.Slot,
		CurrentID:  rID,
		Message:    TmpMsg,
		CurrentKey: cyclic.NewInt(1),
	})
	t.Logf("RTIDENTIFY:\n  AssociatedData: %s\n",
		esTmp.EncryptedRecipient.Text(10))
	expectedRTIdentify := []*cyclic.Int{
		cyclic.NewInt(1),
	}
	if esTmp.EncryptedRecipient.Cmp(expectedRTIdentify[0]) != 0 {
		t.Errorf("RTIDENTIFY failed EncryptedAssociatedData. Got: %s Expected: %s",
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
		is := realtime.Slot(*((*iv).(*realtime.Slot)))
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
	esRT := (*rtnRT).(*realtime.Slot)

	t.Logf("RTPEEL:\n  EncryptedMessage: %s\n",
		esRT.Message.Text(10))
	expectedRTPeel := []*cyclic.Int{
		cyclic.NewInt(31),
	}
	if esRT.Message.Cmp(expectedRTPeel[0]) != 0 {
		t.Errorf("RTPEEL failed EncryptedMessage. Got: %s Expected: %s",
			esRT.Message.Text(10), expectedRTPeel[0].Text(10))
	}

	t.Logf("Final Results: Slot: %d, Recipient ID: %q, Message: %s\n",
		esRT.Slot, esRT.CurrentID,
		esRT.Message.Text(10))
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

	go benchmark.RevealStripTranslate(N2Reveal.OutChannel, N2Strip.InChannel)
	go benchmark.EncryptRevealTranslate(N2Encrypt.OutChannel, N1Reveal.InChannel,
		Node2Round)
	go benchmark.PermuteEncryptTranslate(N2Permute.OutChannel, N1Encrypt.InChannel,
		Node2Round)

	// Run Generate
	genMsg := services.Slot(&precomputation.SlotGeneration{Slot: 0})
	N1Generation.InChannel <- &genMsg
	_ = <-N1Generation.OutChannel
	N2Generation.InChannel <- &genMsg
	_ = <-N2Generation.OutChannel

	t.Logf("2 NODE GENERATION RESULTS: \n")
	t.Logf("%v", benchmark.RoundText(&grp, Node1Round))
	t.Logf("%v", benchmark.RoundText(&grp, Node2Round))

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
	t.Logf("%v", benchmark.RoundText(&grp, Node2Round))
	t.Logf("%v", benchmark.RoundText(&grp, Node1Round))

	// Now finish precomputation
	decMsg := services.Slot(&precomputation.PrecomputationSlot{
		Slot:                      0,
		MessageCypher:             cyclic.NewInt(1),
		MessagePrecomputation:     cyclic.NewInt(1),
		AssociatedDataCypher:         cyclic.NewInt(1),
		AssociatedDataPrecomputation: cyclic.NewInt(1),
	})
	N1Decrypt.InChannel <- &decMsg
	rtn := <-N2Strip.OutChannel
	es := (*rtn).(*precomputation.PrecomputationSlot)

	Node2Round.LastNode.MessagePrecomputation[es.Slot] = es.MessagePrecomputation
	Node2Round.LastNode.RecipientPrecomputation[es.Slot] =
		es.AssociatedDataPrecomputation
	t.Logf("2 NODE STRIP:\n  MessagePrecomputation: %s, "+
		"RecipientPrecomputation: %s\n", es.MessagePrecomputation.Text(10),
		es.AssociatedDataPrecomputation.Text(10))

	// Check precomputation
	MP, RP := ComputePrecomputation(&grp,
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

	go benchmark.RTEncryptRTEncryptTranslate(N1RTEncrypt.OutChannel,
		N2RTEncrypt.InChannel)
	go benchmark.RTDecryptRTDecryptTranslate(N1RTDecrypt.OutChannel,
		N2RTDecrypt.InChannel)
	go benchmark.RTDecryptRTPermuteTranslate(N2RTDecrypt.OutChannel,
		N1RTPermute.InChannel)
	go benchmark.RTPermuteRTIdentifyTranslate(N2RTPermute.OutChannel,
		N2RTIdentify.InChannel, IntermediateMsgs)
	go benchmark.RTIdentifyRTEncryptTranslate(N2RTIdentify.OutChannel,
		N1RTEncrypt.InChannel, IntermediateMsgs)
	go benchmark.RTEncryptRTPeelTranslate(N2RTEncrypt.OutChannel,
		N2RTPeel.InChannel)

	inputMsg := services.Slot(&realtime.Slot{
		Slot:               0,
		CurrentID:          id.NewUserFromUint(1, t),
		Message:            cyclic.NewInt(42), // Meaning of Life
		EncryptedRecipient: cyclic.NewInt(1),
		CurrentKey:         cyclic.NewInt(1),
	})
	N1RTDecrypt.InChannel <- &inputMsg
	rtnRT := <-N2RTPeel.OutChannel
	esRT := (*rtnRT).(*realtime.Slot)
	t.Logf("RTPEEL:\n  EncryptedMessage: %s\n",
		esRT.Message.Text(10))
	expectedRTPeel := []*cyclic.Int{
		cyclic.NewInt(42),
	}
	if esRT.Message.Cmp(expectedRTPeel[0]) != 0 {
		t.Errorf("RTPEEL failed EncryptedMessage. Got: %s Expected: %s",
			esRT.Message.Text(10), expectedRTPeel[0].Text(10))
	}

	t.Logf("Final Results: Slot: %d, Recipient ID: %q, Message: %s\n",
		esRT.Slot, *esRT.CurrentID,
		esRT.Message.Text(10))
}

// Perform an end to end test of the precomputation with batchsize 1,
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
	rng := cyclic.NewRandom(cyclic.NewInt(0), cyclic.NewInt(1000))
	grp := cyclic.NewGroup(cyclic.NewInt(101), cyclic.NewInt(5), cyclic.NewInt(4),
		rng)
	inputMsgs := make([]realtime.Slot, BatchSize)
	outputMsgs := make([]realtime.Slot, BatchSize)
	for i := uint64(0); i < BatchSize; i++ {
		inputMsgs[i] = realtime.Slot{
			Slot:               i,
			CurrentID:          id.NewUserFromUint(i+1, t),
			Message:            cyclic.NewInt(42 + int64(i)), // Meaning of Life
			EncryptedRecipient: cyclic.NewInt(1 + int64(i)),
			CurrentKey:         cyclic.NewInt(1),
		}
		outputMsgs[i] = realtime.Slot{
			Slot:      i,
			CurrentID: id.NewUserFromUint(i+1, t),
			Message:   cyclic.NewInt(42 + int64(i)), // Meaning of Life
		}
	}
	rounds := benchmark.GenerateRounds(nodeCount, BatchSize, &grp)
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
	inputMsgs := make([]realtime.Slot, BatchSize)
	outputMsgs := make([]realtime.Slot, BatchSize)
	for i := uint64(0); i < BatchSize; i++ {
		inputMsgs[i] = realtime.Slot{
			Slot:               i,
			CurrentID:          id.NewUserFromUint(i+1, t),
			Message:            cyclic.NewInt((42 + int64(i)) % 101), // Meaning of Life
			EncryptedRecipient: cyclic.NewInt((1 + int64(i)) % 101),
			CurrentKey:         cyclic.NewInt(1),
		}
		outputMsgs[i] = realtime.Slot{
			Slot:      i,
			CurrentID: id.NewUserFromUint((i+1)%101, t),
			Message:   cyclic.NewInt((42 + int64(i)) % 101), // Meaning of Life
		}
	}
	rounds := benchmark.GenerateRounds(nodeCount, BatchSize, &grp)
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

	rng := cyclic.NewRandom(cyclic.NewInt(0),
		cyclic.NewIntFromString(benchmark.MAXGENERATION, 16))
	grp := cyclic.NewGroup(prime, cyclic.NewInt(5), cyclic.NewInt(4),
		rng)
	inputMsgs := make([]realtime.Slot, BatchSize)
	outputMsgs := make([]realtime.Slot, BatchSize)
	for i := uint64(0); i < BatchSize; i++ {
		inputMsgs[i] = realtime.Slot{
			Slot:               i,
			CurrentID:          id.NewUserFromUint(i+1, t),
			Message:            cyclic.NewInt((42 + int64(i)) % 101), // Meaning of Life
			EncryptedRecipient: cyclic.NewInt((1 + int64(i)) % 101),
			CurrentKey:         cyclic.NewInt(1),
		}
		outputMsgs[i] = realtime.Slot{
			Slot:      i,
			CurrentID: id.NewUserFromUint((i+1)%101, t),
			Message:   cyclic.NewInt((42 + int64(i)) % 101), // Meaning of Life
		}
	}
	rounds := benchmark.GenerateRounds(nodeCount, BatchSize, &grp)
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

	MultiNodeTest(nodeCount, BatchSize, &grp, rounds, inputMsgs, outputMsgs, t)
}

// Call the main benchmark tests so we get coverage for it
func TestBMPrecomp_1_1(b *testing.T)  { benchmark.PrecompIterations(1, 1, 1) }
func TestBMRealtime_1_1(b *testing.T) { benchmark.RealtimeIterations(1, 1, 1) }
