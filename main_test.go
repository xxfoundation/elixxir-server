package main

import (
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/server/node"
	"gitlab.com/privategrity/server/services"
	"gitlab.com/privategrity/server/cryptops/precomputation"
	"gitlab.com/privategrity/server/cryptops/realtime"
	"fmt"
	"testing"
	"strconv"
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

	// We are the last node, so allocate the arrays for LastNode
	node.InitLastNode(round)

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
	var shareMsg services.Slot
	shareMsg = &precomputation.SlotShare{Slot: 0,
				PartialRoundPublicCypherKey: cyclic.NewInt(3)}
	Share := services.DispatchCryptop(&grp, precomputation.Share{}, nil, nil,
		round)
	Share.InChannel <- &shareMsg
	shareResultSlot := <- Share.OutChannel
	shareResult := (*shareResultSlot).(*precomputation.SlotShare)
	round.CypherPublicKey = shareResult.PartialRoundPublicCypherKey
	t.Errorf("Got: %v", round.CypherPublicKey.Text(10))

	// DECRYPT PHASE
	var decMsg services.Slot
	decMsg = &precomputation.SlotDecrypt{
		Slot: 0,
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
		iv := <- in
		is := precomputation.SlotPermute(*((*iv).(*precomputation.SlotDecrypt)))
		ov := services.Slot(&is)
		out <- &ov
	}(Decrypt.OutChannel, Permute.InChannel)

	// // ENCRYPT PHASE
	Encrypt := services.DispatchCryptop(&grp, precomputation.Encrypt{},
		nil, nil, round)

	go func(in, out chan *services.Slot) {
		iv := <- in
		pm := (*iv).(*precomputation.SlotPermute)
		se := precomputation.SlotEncrypt{
			Slot: pm.Slot,
			EncryptedMessageKeys: pm.EncryptedMessageKeys,
			PartialMessageCypherText: pm.PartialMessageCypherText,
		}
		// Save the results to LastNode, which we don't have to check
		// because we are the only node
		i := pm.Slot
		round.LastNode.RecipientCypherText[i] = pm.PartialRecipientIDCypherText
		round.LastNode.EncryptedRecipientPrecomputation[i] = pm.EncryptedRecipientIDKeys
		ov := services.Slot(&se)
		out <- &ov
	}(Permute.OutChannel, Encrypt.InChannel)

	// REVEAL PHASE
	Reveal := services.DispatchCryptop(&grp, precomputation.Reveal{},
		nil, nil, round)

	go func(in, out chan *services.Slot) {
		iv := <- in
		pm := (*iv).(*precomputation.SlotEncrypt)
		i := pm.Slot
		se := precomputation.SlotReveal{
			Slot: i,
			PartialMessageCypherText: pm.PartialMessageCypherText,
			PartialRecipientCypherText: round.LastNode.RecipientCypherText[i],
		}
		// Save the results to LastNode
		round.LastNode.EncryptedMessagePrecomputation[i] = pm.EncryptedMessageKeys
		ov := services.Slot(&se)
		out <- &ov
	}(Encrypt.OutChannel, Reveal.InChannel)

	// STRIP PHASE
	Strip := services.DispatchCryptop(&grp, precomputation.Strip{},
		nil, nil, round)

	go func(in, out chan *services.Slot) {
		iv := <- in
		pm := (*iv).(*precomputation.SlotReveal)
		i := pm.Slot
		se := precomputation.SlotStripIn{
			Slot: i,
			RoundMessagePrivateKey: pm.PartialMessageCypherText,
			RoundRecipientPrivateKey: pm.PartialRecipientCypherText,
		}
		ov := services.Slot(&se)
		out <- &ov
	}(Reveal.OutChannel, Strip.InChannel)


	// KICK OFF PRECOMPUTATION and save
	Decrypt.InChannel <- &decMsg
	rtn := <-Strip.OutChannel
	es := (*rtn).(*precomputation.SlotStripOut)
	fmt.Println("%d, %s, %s",
		es.Slot, es.MessagePrecomputation.Text(10),
		es.RecipientPrecomputation.Text(10))
	round.LastNode.MessagePrecomputation[es.Slot] = es.MessagePrecomputation
	round.LastNode.RecipientPrecomputation[es.Slot] = es.RecipientPrecomputation
	t.Errorf("What? %+v", rtn)

	// ----- REALTIME ----- //
	inputMsg := services.Slot(&realtime.SlotDecryptIn{
		Slot: 0,
		SenderID: 1,
		EncryptedMessage: cyclic.NewInt(1),
		EncryptedRecipientID: cyclic.NewInt(1),
		TransmissionKey: cyclic.NewInt(1),
	})

	// DECRYPT PHASE
	RTDecrypt := services.DispatchCryptop(&grp, realtime.Decrypt{},
		nil, nil, round)

	// PERMUTE PHASE
	RTPermute := services.DispatchCryptop(&grp, realtime.Permute{},
		nil, nil, round)

	go func(in, out chan *services.Slot) {
		iv := <- in
		is := (*iv).(*realtime.SlotDecryptOut)
		ov := services.Slot(&realtime.SlotPermute{
			Slot: is.Slot,
			EncryptedMessage: is.EncryptedMessage,
			EncryptedRecipientID: is.EncryptedRecipientID,
		})
		out <- &ov
	}(RTDecrypt.OutChannel, RTPermute.InChannel)

	// IDENTIFY PHASE
	RTIdentify := services.DispatchCryptop(&grp, realtime.Identify{},
		nil, nil, round)

	// FIXME
	RTDecrypt.InChannel <- &inputMsg
	rtnPrm := <- RTPermute.OutChannel
	esPrm := (*rtnPrm).(*realtime.SlotPermute)
	ovPrm := services.Slot(&realtime.SlotIdentify{
			Slot: esPrm.Slot,
			EncryptedRecipientID: esPrm.EncryptedRecipientID,
	})
	TmpMsg := esPrm.EncryptedMessage

	// HACK HACK HACK FIXME FIXME
	RTIdentify.InChannel <- &ovPrm
	rtnTmp := <-RTIdentify.OutChannel
	esTmp := (*rtnTmp).(*realtime.SlotIdentify)
	rID,_ := strconv.ParseUint(esTmp.EncryptedRecipientID.Text(10), 10, 64)
	inputMsgPostID := services.Slot(&realtime.SlotEncryptIn{
		Slot: esTmp.Slot,
		RecipientID: rID,
		EncryptedMessage: TmpMsg,
		ReceptionKey: cyclic.NewInt(1),
	})

	// ENCRYPT PHASE
	RTEncrypt := services.DispatchCryptop(&grp, realtime.Encrypt{},
		nil, nil, round)

	// PEEL PHASE
	RTPeel := services.DispatchCryptop(&grp, realtime.Peel{},
		nil, nil, round)

	go func(in, out chan *services.Slot) {
		iv := <- in
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
