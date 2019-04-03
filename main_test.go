////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package main

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/large"
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
// the U, V keys to make the associated data precomputation.
func ComputeSingleNodePrecomputation(grp *cyclic.Group, round *globals.Round) (
	*cyclic.Int, *cyclic.Int) {
	MP := grp.NewInt(1)

	grp.Mul(MP, round.R_INV[0], MP)
	grp.Mul(MP, round.S_INV[0], MP)
	grp.Mul(MP, round.T_INV[0], MP)

	RP := grp.NewInt(1)

	grp.Mul(RP, round.U_INV[0], RP)
	grp.Mul(RP, round.V_INV[0], RP)

	return MP, RP

}

// Compute Precomputation for N nodes
// NOTE: This does not handle precomputation under permutation, but it will
//       handle multi-node precomputation checks.
func ComputePrecomputation(grp *cyclic.Group, rounds []*globals.Round) (
	*cyclic.Int, *cyclic.Int) {
	MP := grp.NewInt(1)
	RP := grp.NewInt(1)
	for i := range rounds {
		grp.Mul(MP, rounds[i].R_INV[0], MP)
		grp.Mul(MP, rounds[i].S_INV[0], MP)
		grp.Mul(MP, rounds[i].T_INV[0], MP)

		grp.Mul(RP, rounds[i].U_INV[0], RP)
		grp.Mul(RP, rounds[i].V_INV[0], RP)
	}
	return MP, RP
}

// End to end test of the mathematical functions required to "share" 1
// key (i.e., R)
func RootingTest(grp *cyclic.Group, t *testing.T) {

	K1 := grp.NewInt(94)

	Z := grp.NewInt(11)

	Y1 := grp.NewInt(79)

	gZ := grp.NewInt(1)

	gY1 := grp.NewInt(1)

	MSG := grp.NewInt(1)
	CTXT := grp.NewInt(1)

	IVS := grp.NewInt(1)
	gY1c := grp.NewInt(1)

	RSLT := grp.NewInt(1)

	grp.Exp(grp.GetGCyclic(), Z, gZ)
	grp.RootCoprime(gZ, Z, RSLT)

	t.Logf("GENERATOR:\n\texpected: %#v\n\treceived: %#v\n",
		grp.GetGCyclic().Text(10), RSLT.Text(10))

	grp.Exp(grp.GetGCyclic(), Y1, gY1)
	grp.Mul(K1, gY1, MSG)

	grp.Exp(grp.GetGCyclic(), Z, gZ)
	grp.Exp(gZ, Y1, CTXT)

	grp.RootCoprime(CTXT, Z, gY1c)

	grp.Inverse(gY1c, IVS)

	grp.Mul(MSG, IVS, RSLT)

	t.Logf("ROOT TEST:\n\texpected: %#v\n\treceived: %#v",
		gY1.Text(10), gY1c.Text(10))

}

// End to end test of the mathematical functions required to "share" 2 keys
// (i.e., UV)
func RootingTestDouble(grp *cyclic.Group, t *testing.T) {

	K1 := grp.NewInt(94)
	K2 := grp.NewInt(18)

	Z := grp.NewInt(13)

	Y1 := grp.NewInt(87)
	Y2 := grp.NewInt(79)

	gZ := grp.NewInt(1)

	gY1 := grp.NewInt(1)
	gY2 := grp.NewInt(1)

	K2gY2 := grp.NewInt(1)

	gZY1 := grp.NewInt(1)
	gZY2 := grp.NewInt(1)

	K1gY1 := grp.NewInt(1)
	K1K2gY1Y2 := grp.NewInt(1)
	CTXT := grp.NewInt(1)

	IVS := grp.NewInt(1)
	gY1Y2c := grp.NewInt(1)

	RSLT := grp.NewInt(1)

	K1K2 := grp.NewInt(1)

	grp.Exp(grp.GetGCyclic(), Y1, gY1)
	grp.Mul(K1, gY1, K1gY1)

	grp.Exp(grp.GetGCyclic(), Y2, gY2)
	grp.Mul(K2, gY2, K2gY2)

	grp.Mul(K2gY2, K1gY1, K1K2gY1Y2)

	grp.Exp(grp.GetGCyclic(), Z, gZ)

	grp.Exp(gZ, Y1, gZY1)
	grp.Exp(gZ, Y2, gZY2)

	grp.Mul(gZY1, gZY2, CTXT)

	grp.RootCoprime(CTXT, Z, gY1Y2c)

	t.Logf("ROUND ASSOCIATED DATA PRIVATE KEY:\n\t%#v,\n", gY1Y2c.Text(10))

	grp.Inverse(gY1Y2c, IVS)

	grp.Mul(K1K2gY1Y2, IVS, RSLT)

	grp.Mul(K1, K2, K1K2)

	t.Logf("ROOT TEST DOUBLE:\n\texpected: %#v\n\treceived: %#v",
		RSLT.Text(10), K1K2.Text(10))

}

// End to end test of the mathematical functions required to "share" 3 keys
// (i.e., RST)
func RootingTestTriple(grp *cyclic.Group, t *testing.T) {

	K1 := grp.NewInt(26)
	K2 := grp.NewInt(77)
	K3 := grp.NewInt(100)

	Z := grp.NewInt(13)

	Y1 := grp.NewInt(69)
	Y2 := grp.NewInt(81)
	Y3 := grp.NewInt(13)

	gZ := grp.NewInt(1)

	gY1 := grp.NewInt(1)
	gY2 := grp.NewInt(1)
	gY3 := grp.NewInt(1)

	K1gY1 := grp.NewInt(1)
	K2gY2 := grp.NewInt(1)
	K3gY3 := grp.NewInt(1)

	gZY1 := grp.NewInt(1)
	gZY2 := grp.NewInt(1)
	gZY3 := grp.NewInt(1)

	gZY1Y2 := grp.NewInt(1)

	K1K2gY1Y2 := grp.NewInt(1)
	K1K2K3gY1Y2Y3 := grp.NewInt(1)

	CTXT := grp.NewInt(1)

	IVS := grp.NewInt(1)
	gY1Y2Y3c := grp.NewInt(1)

	RSLT := grp.NewInt(1)

	K1K2 := grp.NewInt(1)
	K1K2K3 := grp.NewInt(1)

	grp.Exp(grp.GetGCyclic(), Y1, gY1)
	grp.Mul(K1, gY1, K1gY1)

	grp.Exp(grp.GetGCyclic(), Y2, gY2)
	grp.Mul(K2, gY2, K2gY2)

	grp.Exp(grp.GetGCyclic(), Y3, gY3)
	grp.Mul(K3, gY3, K3gY3)

	grp.Mul(K2gY2, K1gY1, K1K2gY1Y2)
	grp.Mul(K1K2gY1Y2, K3gY3, K1K2K3gY1Y2Y3)

	grp.Exp(grp.GetGCyclic(), Z, gZ)

	grp.Exp(gZ, Y1, gZY1)
	grp.Exp(gZ, Y2, gZY2)
	grp.Exp(gZ, Y3, gZY3)

	grp.Mul(gZY1, gZY2, gZY1Y2)
	grp.Mul(gZY1Y2, gZY3, CTXT)

	grp.RootCoprime(CTXT, Z, gY1Y2Y3c)

	grp.Inverse(gY1Y2Y3c, IVS)

	grp.Mul(K1K2K3gY1Y2Y3, IVS, RSLT)

	grp.Mul(K1, K2, K1K2)
	grp.Mul(K1K2, K3, K1K2K3)

	t.Logf("ROOT TEST TRIPLE:\n\texpected: %#v\n\treceived: %#v",
		RSLT.Text(10), K1K2K3.Text(10))
}

// Perform an end to end test of the precomputation with batch size 1,
// then use it to send the message through a 1-node system to smoke test
// the cryptographic operations.
// NOTE: This test will not use real associated data, i.e., the recipientID value
// is not set in associated data.
// Trying to do this would lead to many changes:
// Firstly because the recipientID is place on bytes 2:33 of 256,
// meaning the Associated Data representation in the group
// would be much bigger than the hardcoded P value of 107
// Secondly, the first byte of the Associated Data is randomly generated,
// so the expected values throughout the pipeline would need to be calculated on the fly
// Not having proper Associated Data is not an issue in this particular test,
// because here only cryptops are chained
// The actual extraction of recipientID from associated data only occurs in transmit
// handlers from the io package
func TestEndToEndCryptops(t *testing.T) {

	// Init, we use a small prime to make it easier to run the numbers
	// when debugging
	batchSize := uint64(1)
	grp := cyclic.NewGroup(large.NewInt(107), large.NewInt(4), large.NewInt(5))
	round := globals.NewRound(batchSize, &grp)
	round.CypherPublicKey = grp.NewInt(3)
	// p=107 -> 7 bits, so exponents can be of 6 bits at most
	// Overwrite default value of round
	round.ExpSize = uint32(6)

	// We are the last node, so allocate the arrays for LastNode
	globals.InitLastNode(round, &grp)

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
	RootingTest(&grp, t)
	RootingTestDouble(&grp, t)
	RootingTestTriple(&grp, t)

	// Overwrite the generated keys. Note the use of Set to make sure the
	// pointers remain unchanged.
	grp.Set(round.Z, grp.NewInt(13))

	grp.Set(round.R[0], grp.NewInt(35))
	grp.Set(round.R_INV[0], grp.NewInt(26))
	grp.Set(round.Y_R[0], grp.NewInt(69))

	grp.Set(round.S[0], grp.NewInt(21))
	grp.Set(round.S_INV[0], grp.NewInt(77))
	grp.Set(round.Y_S[0], grp.NewInt(81))

	grp.Set(round.T[0], grp.NewInt(100))
	grp.Set(round.T_INV[0], grp.NewInt(100))
	grp.Set(round.Y_T[0], grp.NewInt(13))

	grp.Set(round.U[0], grp.NewInt(72))
	grp.Set(round.U_INV[0], grp.NewInt(94))
	grp.Set(round.Y_U[0], grp.NewInt(87))

	grp.Set(round.V[0], grp.NewInt(73))
	grp.Set(round.V_INV[0], grp.NewInt(18))
	grp.Set(round.Y_V[0], grp.NewInt(79))

	// SHARE PHASE
	var shareMsg services.Slot
	shareMsg = &precomputation.SlotShare{Slot: 0,
		PartialRoundPublicCypherKey: grp.GetGCyclic()}
	Share := services.DispatchCryptop(&grp, precomputation.Share{}, nil, nil,
		round)
	Share.InChannel <- &shareMsg
	shareResultSlot := <-Share.OutChannel
	shareResult := (*shareResultSlot).(*precomputation.SlotShare)
	grp.Set(round.CypherPublicKey, shareResult.PartialRoundPublicCypherKey)

	t.Logf("SHARE:\n%v", benchmark.RoundText(&grp, round))

	if shareResult.PartialRoundPublicCypherKey.Cmp(grp.NewInt(69)) != 0 {
		t.Errorf("SHARE failed\n\texpected: %#v\n\treceived: %#v", 20,
			shareResult.PartialRoundPublicCypherKey.Text(10))
	}

	// DECRYPT PHASE
	var decMsg services.Slot
	decMsg = &precomputation.PrecomputationSlot{
		Slot:                         0,
		MessageCypher:                grp.NewInt(1),
		MessagePrecomputation:        grp.NewInt(1),
		AssociatedDataCypher:         grp.NewInt(1),
		AssociatedDataPrecomputation: grp.NewInt(1),
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
			grp.NewInt(5), grp.NewInt(17),
			grp.NewInt(79), grp.NewInt(36),
		}
		if is.MessageCypher.Cmp(expectedDecrypt[0]) != 0 {
			t.Errorf("DECRYPT failed MessageCypher\n\texpected: %#v\n\treceived: %#v",
				expectedDecrypt[0].Text(10), is.MessageCypher.Text(10))
		}
		if is.AssociatedDataCypher.Cmp(expectedDecrypt[1]) != 0 {
			t.Errorf("DECRYPT failed AssociatedDataCypher\n\texpected: %#v\n\treceived: %#v",
				expectedDecrypt[1].Text(10), is.AssociatedDataCypher.Text(10))
		}
		if is.MessagePrecomputation.Cmp(expectedDecrypt[2]) != 0 {
			t.Errorf("DECRYPT failed MessagePrecomputation\n\texpected: %#v\n\treceived: %#v",
				expectedDecrypt[2].Text(10), is.MessagePrecomputation.Text(10))
		}
		if is.AssociatedDataPrecomputation.Cmp(expectedDecrypt[3]) != 0 {
			t.Errorf("DECRYPT failed AssociatedDataPrecomputation\n\texpected: %#v\n\treceived: %#v",
				expectedDecrypt[3].Text(10), is.AssociatedDataPrecomputation.Text(10))
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
			grp.NewInt(23), grp.NewInt(61),
			grp.NewInt(39), grp.NewInt(85),
		}
		if pm.MessageCypher.Cmp(expectedPermute[0]) != 0 {
			t.Errorf("PERMUTE failed MessageCypher\n\texpected: %#v\n\treceived: %#v",
				expectedPermute[0].Text(10), pm.MessageCypher.Text(10))
		}
		if pm.AssociatedDataCypher.Cmp(expectedPermute[1]) != 0 {
			t.Errorf("PERMUTE failed AssociatedDataCypher\n\texpected: %#v\n\treceived: %#v",
				expectedPermute[1].Text(10), pm.AssociatedDataCypher.Text(10))
		}
		if pm.MessagePrecomputation.Cmp(expectedPermute[2]) != 0 {
			t.Errorf("PERMUTE failed MessagePrecomputation\n\texpected: %#v\n\treceived: %#v",
				expectedPermute[2].Text(10), pm.MessagePrecomputation.Text(10))
		}
		if pm.AssociatedDataPrecomputation.Cmp(expectedPermute[3]) != 0 {
			t.Errorf("PERMUTE failed AssociatedDataPrecomputation\n\texpected: %#v\n\treceived: %#v",
				expectedPermute[3].Text(10), pm.AssociatedDataPrecomputation.Text(10))
		}

		// Save the results to LastNode, which we don't have to check
		// because we are the only node
		i := pm.Slot
		grp.Set(round.LastNode.AssociatedDataCypherText[i],
			pm.AssociatedDataPrecomputation)
		grp.Set(round.LastNode.EncryptedAssociatedDataPrecomputation[i],
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
			grp.NewInt(19), grp.NewInt(27),
		}
		if pm.MessageCypher.Cmp(expectedEncrypt[0]) != 0 {
			t.Errorf("ENCRYPT failed MessageCypher\n\texpected: %#v\n\treceived: %#v",
				expectedEncrypt[0].Text(10), pm.MessageCypher.Text(10))
		}
		if pm.MessagePrecomputation.Cmp(expectedEncrypt[1]) != 0 {
			t.Errorf("ENCRYPT failed AssociatedDataCypher\n\texpected: %#v\n\treceived: %#v",
				expectedEncrypt[1].Text(10), pm.MessagePrecomputation.Text(10))
		}

		// Save the results to LastNode
		grp.Set(round.LastNode.EncryptedMessagePrecomputation[i],
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
			"RoundAssociatedDataPrivateKey: %s\n", pm.AssociatedDataPrecomputation.Text(10),
			pm.AssociatedDataPrecomputation.Text(10))
		expectedReveal := []*cyclic.Int{
			grp.NewInt(42), grp.NewInt(13),
		}
		if pm.MessagePrecomputation.Cmp(expectedReveal[0]) != 0 {
			t.Errorf("REVEAL failed RoundMessagePrivateKey\n\texpected: %#v\n\treceived: %#v",
				expectedReveal[0].Text(10), pm.MessagePrecomputation.Text(10))
		}
		if pm.AssociatedDataPrecomputation.Cmp(expectedReveal[1]) != 0 {
			t.Errorf("REVEAL failed RoundAssociatedDataPrivateKey\n\texpected: %#v\n\treceived: %#v",
				expectedReveal[1].Text(10), pm.AssociatedDataPrecomputation.Text(10))
		}

		ov := services.Slot(pm)
		out <- &ov
	}(Reveal.OutChannel, Strip.InChannel)

	// KICK OFF PRECOMPUTATION and save
	Decrypt.InChannel <- &decMsg
	rtn := <-Strip.OutChannel
	es := (*rtn).(*precomputation.PrecomputationSlot)

	round.LastNode.MessagePrecomputation[es.Slot] = es.MessagePrecomputation
	round.LastNode.AssociatedDataPrecomputation[es.Slot] = es.AssociatedDataPrecomputation

	t.Logf("STRIP:\n  MessagePrecomputation: %s, "+
		"AssociatedDataPrecomputation: %s\n", es.MessagePrecomputation.Text(10),
		es.AssociatedDataPrecomputation.Text(10))
	expectedStrip := []*cyclic.Int{
		grp.NewInt(3), grp.NewInt(87),
	}
	if es.MessagePrecomputation.Cmp(expectedStrip[0]) != 0 {
		t.Errorf("STRIP failed MessagePrecomputation\n\texpected: %#v\n\treceived: %#v",
			expectedStrip[0].Text(10), es.MessagePrecomputation.Text(10))
	}
	if es.AssociatedDataPrecomputation.Cmp(expectedStrip[1]) != 0 {
		t.Errorf("STRIP failed AssociatedDataPrecomputation\n\texpected: %#v\n\treceived: %#v",
			expectedStrip[1].Text(10), es.AssociatedDataPrecomputation.Text(10))
	}

	MP, RP := ComputeSingleNodePrecomputation(&grp, round)

	if MP.Cmp(es.MessagePrecomputation) != 0 {
		t.Errorf("Message Precomputation Incorrect!\n\texpected: %#v\n\treceived: %#v",
			MP.Text(10), es.MessagePrecomputation.Text(10))
	}

	if RP.Cmp(es.AssociatedDataPrecomputation) != 0 {
		t.Errorf("Associated Data Precomputation Incorrect!\n\texpected: %#v\n\treceived: %#v",
			RP.Text(10), es.AssociatedDataPrecomputation.Text(10))
	}

	// ----- REALTIME ----- //
	inputMsg := services.Slot(&realtime.Slot{
		Slot:           0,
		CurrentID:      id.NewUserFromUint(1, t),
		Message:        grp.NewInt(31),
		AssociatedData: grp.NewInt(1),
		CurrentKey:     grp.NewInt(1),
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
			Slot:           is.Slot,
			Message:        is.Message,
			AssociatedData: is.AssociatedData,
		})

		t.Logf("RTDECRYPT:\n  EncryptedMessage: %s, AssociatedData: %s\n",
			is.Message.Text(10),
			is.AssociatedData.Text(10))
		expectedRTDecrypt := []*cyclic.Int{
			grp.NewInt(15), grp.NewInt(72),
		}
		if is.Message.Cmp(expectedRTDecrypt[0]) != 0 {
			t.Errorf("RTDECRYPT failed EncryptedMessage\n\texpected: %#v\n\treceived: %#v",
				expectedRTDecrypt[0].Text(10), is.Message.Text(10))
		}
		if is.AssociatedData.Cmp(expectedRTDecrypt[1]) != 0 {
			t.Errorf("RTDECRYPT failed AssociatedData\n\texpected: %#v\n\treceived: %#v",
				expectedRTDecrypt[1].Text(10), is.AssociatedData.Text(10))
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
		Slot:           esPrm.Slot,
		AssociatedData: esPrm.AssociatedData,
	})
	TmpMsg := esPrm.Message
	t.Logf("RTPERMUTE:\n  AssociatedData: %s\n",
		esPrm.AssociatedData.Text(10))
	expectedRTPermute := []*cyclic.Int{
		grp.NewInt(13),
	}
	if esPrm.AssociatedData.Cmp(expectedRTPermute[0]) != 0 {
		t.Errorf("RTPERMUTE failed AssociatedData. Got: %s Expected: %s",
			esPrm.AssociatedData.Text(10), expectedRTPermute[0].Text(10))
	}

	RTIdentify.InChannel <- &ovPrm
	rtnTmp := <-RTIdentify.OutChannel
	esTmp := (*rtnTmp).(*realtime.Slot)
	rID := new(id.User).SetBytes(esTmp.AssociatedData.
		LeftpadBytes(id.UserLen))
	copy(rID[:], esTmp.AssociatedData.LeftpadBytes(id.UserLen))
	inputMsgPostID := services.Slot(&realtime.Slot{
		Slot:       esTmp.Slot,
		CurrentID:  rID,
		Message:    TmpMsg,
		CurrentKey: grp.NewInt(1),
	})
	t.Logf("RTIDENTIFY:\n  AssociatedData: %s\n",
		esTmp.AssociatedData.Text(10))
	expectedRTIdentify := []*cyclic.Int{
		grp.NewInt(61),
	}
	if esTmp.AssociatedData.Cmp(expectedRTIdentify[0]) != 0 {
		t.Errorf("RTIDENTIFY failed AssociatedData. Got: %s Expected: %s",
			esTmp.AssociatedData.Text(10), expectedRTIdentify[0].Text(10))
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
			grp.NewInt(42),
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
		grp.NewInt(19),
	}
	if esRT.Message.Cmp(expectedRTPeel[0]) != 0 {
		t.Errorf("RTPEEL failed EncryptedMessage. Got: %s Expected: %s",
			esRT.Message.Text(10), expectedRTPeel[0].Text(10))
	}

	t.Logf("Final Results: Slot: %d, Recipient ID: %q, Message: %s\n",
		esRT.Slot, esRT.CurrentID,
		esRT.Message.Text(10))
}

// Perform an end to end test of the precomputation with batch size 1,
// then use it to send the message through a 2-node system to smoke test
// the cryptographic operations.
func TestEndToEndCryptopsWith2Nodes(t *testing.T) {

	// Init, we use a small prime to make it easier to run the numbers
	// when debugging
	batchSize := uint64(1)
	grp := cyclic.NewGroup(large.NewInt(107), large.NewInt(4), large.NewInt(5))
	Node1Round := globals.NewRound(batchSize, &grp)
	Node2Round := globals.NewRound(batchSize, &grp)
	Node1Round.CypherPublicKey = grp.NewInt(1)
	Node2Round.CypherPublicKey = grp.NewInt(1)

	// p=107 -> 7 bits, so exponents can be of 6 bits at most
	// Overwrite default value of rounds
	Node1Round.ExpSize = uint32(6)
	Node2Round.ExpSize = uint32(6)

	// Allocate the arrays for LastNode
	globals.InitLastNode(Node2Round, &grp)

	// ----- PRECOMPUTATION ----- //
	N1Generation := services.DispatchCryptop(&grp, precomputation.Generation{},
		nil, nil, Node1Round)
	N2Generation := services.DispatchCryptop(&grp, precomputation.Generation{},
		nil, nil, Node2Round)
	// Since round.Z is generated on creation of the Generation precomp,
	// need to loop the generation here until a valid Z is produced
	maxInt := grp.NewMaxInt()
	for Node1Round.Z.Cmp(maxInt) == 0 || Node2Round.Z.Cmp(maxInt) == 0 {
		N1Generation = services.DispatchCryptop(&grp, precomputation.Generation{},
			nil, nil, Node1Round)
		N2Generation = services.DispatchCryptop(&grp, precomputation.Generation{},
			nil, nil, Node2Round)
	}

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
		Node2Round, &grp)
	go benchmark.PermuteEncryptTranslate(N2Permute.OutChannel, N1Encrypt.InChannel,
		Node2Round, &grp)

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
		PartialRoundPublicCypherKey: grp.GetGCyclic()})
	N1Share.InChannel <- &shareMsg
	shareResultSlot := <-N2Share.OutChannel
	shareResult := (*shareResultSlot).(*precomputation.SlotShare)
	PublicCypherKey := grp.NewInt(1)
	grp.Set(PublicCypherKey, shareResult.PartialRoundPublicCypherKey)
	grp.Set(Node1Round.CypherPublicKey, PublicCypherKey)
	grp.Set(Node2Round.CypherPublicKey, PublicCypherKey)

	t.Logf("2 NODE SHARE RESULTS: \n")
	t.Logf("%v", benchmark.RoundText(&grp, Node2Round))
	t.Logf("%v", benchmark.RoundText(&grp, Node1Round))

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
	IntermediateMsgs[0] = grp.NewInt(1)

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
		N2RTEncrypt.InChannel, &grp)
	go benchmark.RTDecryptRTDecryptTranslate(N1RTDecrypt.OutChannel,
		N2RTDecrypt.InChannel, &grp)
	go benchmark.RTDecryptRTPermuteTranslate(N2RTDecrypt.OutChannel,
		N1RTPermute.InChannel)
	go benchmark.RTPermuteRTIdentifyTranslate(N2RTPermute.OutChannel,
		N2RTIdentify.InChannel, IntermediateMsgs, &grp)
	go benchmark.RTIdentifyRTEncryptTranslate(N2RTIdentify.OutChannel,
		N1RTEncrypt.InChannel, IntermediateMsgs, &grp)
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
	rounds := benchmark.GenerateRounds(nodeCount, BatchSize, &grp)
	MultiNodeTest(nodeCount, BatchSize, &grp, rounds, inputMsgs, outputMsgs, t)
}

func Test1NodePermuteE2E(t *testing.T) {
	nodeCount := 1
	BatchSize := uint64(100)

	primeStrng := "107"

	prime := large.NewInt(0)
	prime.SetString(primeStrng, 10)

	grp := cyclic.NewGroup(prime, large.NewInt(4), large.NewInt(5))
	inputMsgs := make([]realtime.Slot, BatchSize)
	outputMsgs := make([]realtime.Slot, BatchSize)
	for i := uint64(0); i < BatchSize; i++ {
		inputMsgs[i] = realtime.Slot{
			Slot:           i,
			CurrentID:      id.NewUserFromUint(i+1, t),
			Message:        grp.NewInt((42+int64(i))%106 + 1), // Meaning of Life
			AssociatedData: grp.NewInt((1+int64(i))%106 + 1),
			CurrentKey:     grp.NewInt(1),
		}
		outputMsgs[i] = realtime.Slot{
			Slot:      i,
			CurrentID: id.NewUserFromUint((i+1)%106+1, t),
			Message:   grp.NewInt((42+int64(i))%106 + 1), // Meaning of Life
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
