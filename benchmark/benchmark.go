////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// benchmark runs parameterized benchmarking simulations of the server
package benchmark

import (
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/server/cryptops/precomputation"
	"gitlab.com/privategrity/server/cryptops/realtime"
	"gitlab.com/privategrity/server/globals"
	"gitlab.com/privategrity/server/services"

	jww "github.com/spf13/jwalterweatherman"

	"fmt"
	"strconv"
)

var PRIME = "FFFFFFFFFFFFFFFFC90FDAA22168C234C4C6628B80DC1CD1" +
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

var MAXGENERATION = "FFFFFFFFFFFFFFFFC90FDAA22168C234C4C6628B80DC1CD1" +
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
	"FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF"

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

// Helper function to initialize round keys. Useful when you only need to edit 1
// element (e.g., the Permutation) in the set of keys held in round
func GenerateRounds(nodeCount int, BatchSize uint64,
	group *cyclic.Group) []*globals.Round {
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

	// TODO: Consider moving this to the callers where it matters
	// t.Logf("%d NODE GENERATION RESULTS: \n", nodeCount)
	// for i := 0; i < nodeCount; i++ {
	// 	t.Logf("%v", RoundText(group, rounds[i]))
	// }

	return rounds
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

func MultiNodePrecomp(nodeCount int, BatchSize uint64,
	group *cyclic.Group, rounds []*globals.Round) {
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
			if nodeCount == 1 {
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
			} else {
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

	// TODO: Consider moving to caller
	// t.Logf("%d NODE SHARE RESULTS: \n", nodeCount)
	// for i := 0; i < nodeCount; i++ {
	// 	t.Logf("%v", RoundText(group, rounds[i]))
	// }

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

		// TOOD: Consider moving this to the caller
		// t.Logf("%d NODE STRIP:\n  MessagePrecomputation: %s, "+
		// 	"RecipientPrecomputation: %s\n", nodeCount,
		// 	es.MessagePrecomputation.Text(10),
		// 	es.RecipientIDPrecomputation.Text(10))

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
	inputMsgs []realtime.RealtimeSlot, expectedOutputs []realtime.RealtimeSlot) {

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

	// TODO: Consider doing this better and re-enabling the prints if they are
	//       useful

	for i := uint64(0); i < BatchSize; i++ {
		rtnRT := <-LNRTPeel.OutChannel
		esRT := (*rtnRT).(*realtime.RealtimeSlot)
		// t.Logf("RTPEEL:\n  EncryptedMessage: %s\n",
		// 	esRT.Message.Text(10))

		if esRT.Message.Cmp(expectedOutputs[i].Message) != 0 {
			jww.FATAL.Panicf("RTPEEL %d failed EncryptedMessage. Got: %s Expected: %s",
				esRT.Slot,
				esRT.Message.Text(10),
				expectedOutputs[i].Message.Text(10))
		}
		if esRT.CurrentID != expectedOutputs[i].CurrentID {
			jww.FATAL.Panicf("RTPEEL %d failed RecipientID. Got: %d Expected: %d",
				esRT.Slot, esRT.CurrentID, expectedOutputs[i].CurrentID)
		}

		// t.Logf("Final Results: Slot: %d, Recipient ID: %d, Message: %s\n",
		// 	esRT.Slot, esRT.CurrentID, esRT.Message.Text(10))
	}

}

func CopyRounds(nodeCount int, r []*globals.Round) []*globals.Round {
	tmp := make([]*globals.Round, nodeCount)
	for i := 0; i < nodeCount; i++ {
		tmp[i] = globals.NewRound(r[i].BatchSize)
		if (i + 1) == nodeCount {
			globals.InitLastNode(tmp[i])
		}
		for j := uint64(0); j < r[i].BatchSize; j++ {
			tmp[i].R[j].Set(r[i].R[j])
			tmp[i].S[j].Set(r[i].S[j])
			tmp[i].T[j].Set(r[i].T[j])
			tmp[i].V[j].Set(r[i].V[j])
			tmp[i].U[j].Set(r[i].U[j])
			tmp[i].R_INV[j].Set(r[i].R_INV[j])
			tmp[i].S_INV[j].Set(r[i].S_INV[j])
			tmp[i].T_INV[j].Set(r[i].T_INV[j])
			tmp[i].V_INV[j].Set(r[i].V_INV[j])
			tmp[i].U_INV[j].Set(r[i].U_INV[j])
			tmp[i].Y_R[j].Set(r[i].Y_R[j])
			tmp[i].Y_S[j].Set(r[i].Y_S[j])
			tmp[i].Y_T[j].Set(r[i].Y_T[j])
			tmp[i].Y_V[j].Set(r[i].Y_V[j])
			tmp[i].Y_U[j].Set(r[i].Y_U[j])
			tmp[i].Permutations[j] = r[i].Permutations[j]
			if (i + 1) == nodeCount {
				tmp[i].LastNode.MessagePrecomputation[j].Set(
					r[i].LastNode.MessagePrecomputation[j])
				tmp[i].LastNode.RecipientPrecomputation[j].Set(
					r[i].LastNode.RecipientPrecomputation[j])
				tmp[i].LastNode.RoundMessagePrivateKey[j].Set(
					r[i].LastNode.RoundMessagePrivateKey[j])
				tmp[i].LastNode.RoundRecipientPrivateKey[j].Set(
					r[i].LastNode.RoundRecipientPrivateKey[j])
				tmp[i].LastNode.RecipientCypherText[j].Set(
					r[i].LastNode.RecipientCypherText[j])
				tmp[i].LastNode.EncryptedRecipientPrecomputation[j].Set(
					r[i].LastNode.EncryptedRecipientPrecomputation[j])
				tmp[i].LastNode.EncryptedMessagePrecomputation[j].Set(
					r[i].LastNode.EncryptedMessagePrecomputation[j])
				tmp[i].LastNode.EncryptedMessage[j].Set(
					r[i].LastNode.EncryptedMessage[j])
			}
		}
		tmp[i].CypherPublicKey.Set(r[i].CypherPublicKey)
		tmp[i].Z.Set(r[i].Z)
	}
	return tmp
}

func GenerateIOMessages(nodeCount int, batchSize uint64,
	rounds []*globals.Round) ([]realtime.RealtimeSlot, []realtime.RealtimeSlot) {
	inputMsgs := make([]realtime.RealtimeSlot, batchSize)
	outputMsgs := make([]realtime.RealtimeSlot, batchSize)
	for i := uint64(0); i < batchSize; i++ {
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
	for i := 0; i < nodeCount; i++ {
		// Now apply  permutations list to outputMsgs
		newOutMsgs := make([]realtime.RealtimeSlot, batchSize)
		for j := uint64(0); j < batchSize; j++ {
			newOutMsgs[rounds[i].Permutations[j]] = outputMsgs[j]
		}

		copy(outputMsgs, newOutMsgs)
		outputMsgs = newOutMsgs
	}

	return inputMsgs, outputMsgs
}

// Template function for running precomputation
func PrecompIterations(nodeCount int, batchSize uint64, iterations int) {
	prime := cyclic.NewInt(0)
	prime.SetString(PRIME, 16)

	rng := cyclic.NewRandom(cyclic.NewInt(0),
		cyclic.NewIntFromString(MAXGENERATION, 16))
	grp := cyclic.NewGroup(prime, cyclic.NewInt(5), cyclic.NewInt(4),
		rng)
	rounds := GenerateRounds(nodeCount, batchSize, &grp)

	for i := 0; i < iterations; i++ {
		MultiNodePrecomp(nodeCount, batchSize, &grp, rounds)
	}
}

// Run realtime simulation for given number of of iterations
func RealtimeIterations(nodeCount int, batchSize uint64, iterations int) {
	prime := cyclic.NewInt(0)
	prime.SetString(PRIME, 16)

	rng := cyclic.NewRandom(cyclic.NewInt(0), cyclic.NewInt(1000))
	grp := cyclic.NewGroup(prime, cyclic.NewInt(5), cyclic.NewInt(4),
		rng)

	rounds := GenerateRounds(nodeCount, batchSize, &grp)

	// Rewrite permutation pattern
	for i := 0; i < nodeCount; i++ {
		for j := uint64(0); j < batchSize; j++ {
			// Shift by 1
			newj := (j + 1) % batchSize
			rounds[i].Permutations[j] = newj
		}
	}

	MultiNodePrecomp(nodeCount, batchSize, &grp, rounds)

	for i := 0; i < iterations; i++ {
		tmpRounds := CopyRounds(nodeCount, rounds)
		inputMsgs, outputMsgs := GenerateIOMessages(nodeCount, batchSize, tmpRounds)

		MultiNodeRealtime(nodeCount, batchSize, &grp, tmpRounds, inputMsgs,
			outputMsgs)
	}
}
