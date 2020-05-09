////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// benchmark runs parameterized benchmarking simulations of the server
package benchmark

import (
	//jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/crypto/cyclic"
	//"gitlab.com/elixxir/crypto/large"
	//	"gitlab.com/elixxir/server/cryptops/precomputation"
	//	"gitlab.com/elixxir/server/cryptops/realtime"
	//"gitlab.com/elixxir/server/globals"
	//"gitlab.com/elixxir/server/services"
	//"fmt"
	//"gitlab.com/elixxir/primitives/id"
)

var group *cyclic.Group

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

/*
// Convert the round object into a string we can print
func RoundText(g *cyclic.Group, n *globals.Round) string {
	outStr := fmt.Sprintf("\tPrime: 107, Generator: %s, CypherPublicKey: %s, "+
		"Z: %s\n", g.GetG().Text(10), n.CypherPublicKey.Text(10), n.Z.Text(10))
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
	grp *cyclic.Group) []*globals.Round {
	group = grp
	rounds := make([]*globals.Round, nodeCount)
	for i := 0; i < nodeCount; i++ {
		rounds[i] = globals.NewRound(BatchSize, grp)
		rounds[i].CypherPublicKey = grp.NewInt(1)

		// Overwrite default value of rounds ExpSize if group prime is small
		expSize := grp.GetP().BitLen() - 1
		if expSize < 256 {
			rounds[i].ExpSize = uint32(expSize)
		}

		// Last Node initialization
		if i == (nodeCount - 1) {
			globals.InitLastNode(rounds[i], grp)
		}
	}

	maxInt := grp.NewMaxInt()
	// Run the GENERATION step
	generations := make([]*services.ThreadController, nodeCount)
	for i := 0; i < nodeCount; i++ {
		// Since round.Z is generated on creation of the Generation precomp,
		// need to loop the generation here until a valid Z is produced
		// This will only happen for small groups
		for rounds[i].Z.Cmp(maxInt) == 0 || rounds[i].Z.Cmp(maxInt) == 0 {
			generations[i] = services.DispatchCryptop(grp,
				precomputation.Generation{}, nil, nil, rounds[i])
		}
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
	round *globals.Round, grp *cyclic.Group) {
	for permuteSlot := range permute {
		is := (*permuteSlot).(*precomputation.PrecomputationSlot)
		se := services.Slot(&precomputation.PrecomputationSlot{
			Slot:                  is.Slot,
			MessageCypher:         is.MessageCypher,
			MessagePrecomputation: is.MessagePrecomputation,
		})
		// Save LastNode Data to Round
		i := is.Slot
		grp.Set(round.LastNode.AssociatedDataCypherText[i], is.AssociatedDataPrecomputation)
		grp.Set(round.LastNode.EncryptedAssociatedDataPrecomputation[i], is.AssociatedDataCypher)
		encrypt <- &se
	}
}

// Convert Encrypt output slot to Reveal input slot
func EncryptRevealTranslate(encrypt, reveal chan *services.Slot,
	round *globals.Round, grp *cyclic.Group) {
	for encryptSlot := range encrypt {
		is := (*encryptSlot).(*precomputation.PrecomputationSlot)
		i := is.Slot
		sr := services.Slot(&precomputation.PrecomputationSlot{
			Slot:                         i,
			MessagePrecomputation:        is.MessagePrecomputation,
			AssociatedDataPrecomputation: round.LastNode.AssociatedDataCypherText[i],
		})
		grp.Set(round.LastNode.EncryptedMessagePrecomputation[i], is.MessageCypher)
		reveal <- &sr
	}
}

// Convert Reveal output slot to Strip input slot
func RevealStripTranslate(reveal, strip chan *services.Slot) {
	for revealSlot := range reveal {
		is := (*revealSlot).(*precomputation.PrecomputationSlot)
		i := is.Slot
		ss := services.Slot(&precomputation.PrecomputationSlot{
			Slot:                         i,
			MessagePrecomputation:        is.MessagePrecomputation,
			AssociatedDataPrecomputation: is.AssociatedDataPrecomputation,
		})
		strip <- &ss
	}
}

// Convert RTDecrypt output slot to RTPermute input slot
func RTDecryptRTPermuteTranslate(decrypt, permute chan *services.Slot) {
	for decryptSlot := range decrypt {
		is := (*decryptSlot).(*realtime.Slot)
		ov := services.Slot(&realtime.Slot{
			Slot:           is.Slot,
			Message:        is.Message,
			AssociatedData: is.AssociatedData,
		})
		permute <- &ov
	}
}

func RTPermuteRTIdentifyTranslate(permute, identify chan *services.Slot,
	outMsgs []*cyclic.Int, grp *cyclic.Group) {
	for permuteSlot := range permute {
		esPrm := (*permuteSlot).(*realtime.Slot)
		ovPrm := services.Slot(&realtime.Slot{
			Slot:           esPrm.Slot,
			AssociatedData: esPrm.AssociatedData,
		})
		grp.Set(outMsgs[esPrm.Slot], esPrm.Message)
		identify <- &ovPrm
	}
}

func RTIdentifyRTEncryptTranslate(identify, encrypt chan *services.Slot,
	inMsgs []*cyclic.Int, grp *cyclic.Group) {
	for identifySlot := range identify {
		esTmp := (*identifySlot).(*realtime.Slot)
		// TODO this will need to eventually be changed to be the actual
		// extraction of RID from associated data
		// HOWEVER, this will significantly change the main test using this
		// benchmark function, as was commented in: TestEndToEndCryptops
		rID := new(id.ID).SetBytes(esTmp.AssociatedData.
			LeftpadBytes(id.UserLen))

		inputMsgPostID := services.Slot(&realtime.Slot{
			Slot:       esTmp.Slot,
			CurrentID:  rID,
			Message:    inMsgs[esTmp.Slot],
			CurrentKey: grp.NewInt(1),
		})
		encrypt <- &inputMsgPostID
	}
}

func RTEncryptRTPeelTranslate(encrypt, peel chan *services.Slot) {
	for encryptSlot := range encrypt {
		is := realtime.Slot(*((*encryptSlot).(*realtime.Slot)))
		ov := services.Slot(&is)
		peel <- &ov
	}
}

func RTDecryptRTDecryptTranslate(in, out chan *services.Slot, grp *cyclic.Group) {
	for is := range in {
		o := (*is).(*realtime.Slot)
		os := services.Slot(&realtime.Slot{
			Slot:           o.Slot,
			CurrentID:      o.CurrentID,
			Message:        o.Message,
			AssociatedData: o.AssociatedData,
			CurrentKey:     grp.NewInt(1), // WTF? FIXME
		})
		out <- &os
	}
}

func RTEncryptRTEncryptTranslate(in, out chan *services.Slot, grp *cyclic.Group) {
	for is := range in {
		o := (*is).(*realtime.Slot)
		os := services.Slot(&realtime.Slot{
			Slot:       o.Slot,
			CurrentID:  o.CurrentID,
			Message:    o.Message,
			CurrentKey: grp.NewInt(1), // FIXME
		})
		out <- &os
	}
}

func MultiNodePrecomp(nodeCount int, BatchSize uint64,
	grp *cyclic.Group, rounds []*globals.Round) {
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
				shares[i] = services.DispatchCryptop(grp, precomputation.Share{},
					nil, nil, rounds[i])
				decrypts[i] = services.DispatchCryptop(grp, precomputation.Decrypt{},
					nil, decrPerm, rounds[i])
				permutes[i] = services.DispatchCryptop(grp, precomputation.Permute{},
					decrPerm, nil, rounds[i])
				encrypts[i] = services.DispatchCryptop(grp, precomputation.Encrypt{},
					nil, nil, rounds[i])
				reveals[i] = services.DispatchCryptop(grp, precomputation.Reveal{},
					nil, nil, rounds[i])
			} else {
				shares[i] = services.DispatchCryptop(grp, precomputation.Share{},
					nil, nil, rounds[i])
				decrypts[i] = services.DispatchCryptop(grp, precomputation.Decrypt{},
					nil, nil, rounds[i])
				permutes[i] = services.DispatchCryptop(grp, precomputation.Permute{},
					decrPerm, nil, rounds[i])
				encrypts[i] = services.DispatchCryptop(grp, precomputation.Encrypt{},
					nil, nil, rounds[i])
				reveals[i] = services.DispatchCryptop(grp, precomputation.Reveal{},
					nil, nil, rounds[i])
			}

		} else if i < (nodeCount - 1) {
			shares[i] = services.DispatchCryptop(grp, precomputation.Share{},
				shares[i-1].OutChannel, nil, rounds[i])
			decrypts[i] = services.DispatchCryptop(grp, precomputation.Decrypt{},
				decrypts[i-1].OutChannel, nil, rounds[i])
			permutes[i] = services.DispatchCryptop(grp, precomputation.Permute{},
				permutes[i-1].OutChannel, nil, rounds[i])
			encrypts[i] = services.DispatchCryptop(grp, precomputation.Encrypt{},
				encrypts[i-1].OutChannel, nil, rounds[i])
			reveals[i] = services.DispatchCryptop(grp, precomputation.Reveal{},
				reveals[i-1].OutChannel, nil, rounds[i])
		} else {
			shares[i] = services.DispatchCryptop(grp, precomputation.Share{},
				shares[i-1].OutChannel, nil, rounds[i])
			decrypts[i] = services.DispatchCryptop(grp, precomputation.Decrypt{},
				decrypts[i-1].OutChannel, decrPerm, rounds[i])
			permutes[i] = services.DispatchCryptop(grp, precomputation.Permute{},
				permutes[i-1].OutChannel, nil, rounds[i])
			encrypts[i] = services.DispatchCryptop(grp, precomputation.Encrypt{},
				encrypts[i-1].OutChannel, nil, rounds[i])
			reveals[i] = services.DispatchCryptop(grp, precomputation.Reveal{},
				reveals[i-1].OutChannel, nil, rounds[i])
		}
	}

	LNStrip := services.DispatchCryptop(grp, precomputation.Strip{},
		nil, nil, LastRound)

	go RevealStripTranslate(reveals[nodeCount-1].OutChannel,
		LNStrip.InChannel)
	go EncryptRevealTranslate(encrypts[nodeCount-1].OutChannel,
		reveals[0].InChannel, LastRound, grp)
	go PermuteEncryptTranslate(permutes[nodeCount-1].OutChannel,
		encrypts[0].InChannel, LastRound, grp)
	//go DecryptPermuteTranslate(decrypts[nodeCount-1].OutChannel,
	//	permutes[0].InChannel)

	// Run Share -- Then save the result to both rounds
	// Note that the outchannel for N1Share is the input channel for N2share
	shareMsg := services.Slot(&precomputation.SlotShare{
		PartialRoundPublicCypherKey: grp.GetGCyclic()})
	shares[0].InChannel <- &shareMsg
	shareResultSlot := <-shares[nodeCount-1].OutChannel
	shareResult := (*shareResultSlot).(*precomputation.SlotShare)
	PublicCypherKey := grp.NewInt(1)
	group.Set(PublicCypherKey, shareResult.PartialRoundPublicCypherKey)
	for i := 0; i < nodeCount; i++ {
		group.Set(rounds[i].CypherPublicKey, PublicCypherKey)
	}

	// TODO: Consider moving to caller
	// t.Logf("%d NODE SHARE RESULTS: \n", nodeCount)
	// for i := 0; i < nodeCount; i++ {
	// 	t.Logf("%v", RoundText(grp, rounds[i]))
	// }

	// Now finish precomputation
	for i := uint64(0); i < BatchSize; i++ {
		decMsg := services.Slot(&precomputation.PrecomputationSlot{
			Slot:                         i,
			MessageCypher:                grp.NewInt(1),
			MessagePrecomputation:        grp.NewInt(1),
			AssociatedDataCypher:         grp.NewInt(1),
			AssociatedDataPrecomputation: grp.NewInt(1),
		})
		decrypts[0].InChannel <- &decMsg
	}

	for i := uint64(0); i < BatchSize; i++ {
		rtn := <-LNStrip.OutChannel
		es := (*rtn).(*precomputation.PrecomputationSlot)

		LastRound.LastNode.MessagePrecomputation[es.Slot] = es.MessagePrecomputation
		LastRound.LastNode.AssociatedDataPrecomputation[es.Slot] =
			es.AssociatedDataPrecomputation

		// TODO: Consider moving this to the caller
		// t.Logf("%d NODE STRIP:\n  MessagePrecomputation: %s, "+
		// 	"AssociatedDataPrecomputation: %s\n", nodeCount,
		// 	es.MessagePrecomputation.Text(10),
		// 	es.AssociatedDataPrecomputation.Text(10))

		// Check precomputation, note that these are currently expected to be
		// wrong under permutation
		// MP, RP := ComputePrecomputation(grp, rounds)

		// if MP.Cmp(es.MessagePrecomputation) != 0 {
		// 	t.Logf("Message Precomputation Incorrect! Expected: %s, "+
		// 		"Received: %s\n",
		// 		MP.Text(10), es.MessagePrecomputation.Text(10))
		// }
		// if RP.Cmp(es.AssociatedDataPrecomputation) != 0 {
		// 	t.Logf("Recipient Precomputation Incorrect! Expected: %s,"+
		// 		" Received: %s\n",
		// 		RP.Text(10), es.AssociatedDataPrecomputation.Text(10))
		// }
	}
}

func MultiNodeRealtime(nodeCount int, BatchSize uint64,
	grp *cyclic.Group, rounds []*globals.Round,
	inputMsgs []realtime.Slot, expectedOutputs []realtime.Slot) {

	LastRound := rounds[nodeCount-1]

	// ----- REALTIME ----- //
	IntermediateMsgs := make([]*cyclic.Int, BatchSize)
	for i := uint64(0); i < BatchSize; i++ {
		IntermediateMsgs[i] = grp.NewInt(1)
	}
	rtdecrypts := make([]*services.ThreadController, nodeCount)
	rtpermutes := make([]*services.ThreadController, nodeCount)
	reorgs := make([]*services.ThreadController, nodeCount)
	rtencrypts := make([]*services.ThreadController, nodeCount)
	for i := 0; i < nodeCount; i++ {
		rtdecrypts[i] = services.DispatchCryptop(grp,
			realtime.Decrypt{}, nil, nil, rounds[i])

		// NOTE: Permute -> reorg -> Permute -> ... -> reorg -> Identify
		reorgs[i] = services.NewSlotReorganizer(nil, nil, BatchSize)
		if i == 0 {
			rtpermutes[i] = services.DispatchCryptop(grp,
				realtime.Permute{}, nil, reorgs[i].InChannel, rounds[i])
		} else {
			rtpermutes[i] = services.DispatchCryptop(grp,
				realtime.Permute{}, reorgs[i-1].OutChannel, reorgs[i].InChannel,
				rounds[i])
		}
		rtencrypts[i] = services.DispatchCryptop(grp,
			realtime.Encrypt{}, nil, nil, rounds[i])

		if i != 0 {
			go RTEncryptRTEncryptTranslate(rtencrypts[i-1].OutChannel,
				rtencrypts[i].InChannel, grp)
			go RTDecryptRTDecryptTranslate(rtdecrypts[i-1].OutChannel,
				rtdecrypts[i].InChannel, grp)
		}
	}

	LNRTIdentify := services.DispatchCryptop(grp,
		realtime.Identify{}, nil, nil, LastRound)
	LNRTPeel := services.DispatchCryptop(grp,
		realtime.Peel{}, nil, nil, LastRound)

	go RTDecryptRTPermuteTranslate(rtdecrypts[nodeCount-1].OutChannel,
		rtpermutes[0].InChannel)
	go RTPermuteRTIdentifyTranslate(reorgs[nodeCount-1].OutChannel,
		LNRTIdentify.InChannel, IntermediateMsgs, grp)
	go RTIdentifyRTEncryptTranslate(LNRTIdentify.OutChannel,
		rtencrypts[0].InChannel, IntermediateMsgs, grp)
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
		esRT := (*rtnRT).(*realtime.Slot)
		// t.Logf("RTPEEL:\n  EncryptedMessage: %s\n",
		// 	esRT.Message.Text(10))

		if esRT.Message.Cmp(expectedOutputs[esRT.Slot].Message) != 0 {
			jww.FATAL.Panicf("RTPEEL %d failed EncryptedMessage. Got: %s Expected: %s",
				esRT.Slot,
				esRT.Message.Text(10),
				expectedOutputs[i].Message.Text(10))
		}
		if *esRT.CurrentID != *expectedOutputs[esRT.Slot].CurrentID {
			jww.FATAL.Panicf("RTPEEL %d failed AssociatedData. Got: %q Expected: %q",
				esRT.Slot, *esRT.CurrentID, *expectedOutputs[i].CurrentID)
		}

		// t.Logf("Final Results: Slot: %d, Recipient ID: %d, Message: %s\n",
		// 	esRT.Slot, esRT.CurrentID, esRT.Message.Text(10))
	}

}

func CopyRounds(nodeCount int, r []*globals.Round,
	grp *cyclic.Group) []*globals.Round {
	tmp := make([]*globals.Round, nodeCount)
	for i := 0; i < nodeCount; i++ {
		tmp[i] = globals.NewRound(r[i].BatchSize, grp)

		if (i + 1) == nodeCount {
			globals.InitLastNode(tmp[i], grp)
		}

		for j := uint64(0); j < r[i].BatchSize; j++ {
			group.Set(tmp[i].R[j], r[i].R[j])
			group.Set(tmp[i].S[j], r[i].S[j])
			group.Set(tmp[i].T[j], r[i].T[j])
			group.Set(tmp[i].V[j], r[i].V[j])
			group.Set(tmp[i].U[j], r[i].U[j])
			group.Set(tmp[i].R_INV[j], r[i].R_INV[j])
			group.Set(tmp[i].S_INV[j], r[i].S_INV[j])
			group.Set(tmp[i].T_INV[j], r[i].T_INV[j])
			group.Set(tmp[i].V_INV[j], r[i].V_INV[j])
			group.Set(tmp[i].U_INV[j], r[i].U_INV[j])
			group.Set(tmp[i].Y_R[j], r[i].Y_R[j])
			group.Set(tmp[i].Y_S[j], r[i].Y_S[j])
			group.Set(tmp[i].Y_T[j], r[i].Y_T[j])
			group.Set(tmp[i].Y_V[j], r[i].Y_V[j])
			group.Set(tmp[i].Y_U[j], r[i].Y_U[j])
			tmp[i].Permutations[j] = r[i].Permutations[j]

			if (i + 1) == nodeCount {
				group.Set(tmp[i].LastNode.MessagePrecomputation[j],
					r[i].LastNode.MessagePrecomputation[j])
				group.Set(tmp[i].LastNode.AssociatedDataPrecomputation[j],
					r[i].LastNode.AssociatedDataPrecomputation[j])
				group.Set(tmp[i].LastNode.RoundMessagePrivateKey[j],
					r[i].LastNode.RoundMessagePrivateKey[j])
				group.Set(tmp[i].LastNode.RoundAssociatedDataPrivateKey[j],
					r[i].LastNode.RoundAssociatedDataPrivateKey[j])
				group.Set(tmp[i].LastNode.AssociatedDataCypherText[j],
					r[i].LastNode.AssociatedDataCypherText[j])
				group.Set(tmp[i].LastNode.EncryptedAssociatedDataPrecomputation[j],
					r[i].LastNode.EncryptedAssociatedDataPrecomputation[j])
				group.Set(tmp[i].LastNode.EncryptedMessagePrecomputation[j],
					r[i].LastNode.EncryptedMessagePrecomputation[j])
				group.Set(tmp[i].LastNode.EncryptedMessage[j],
					r[i].LastNode.EncryptedMessage[j])
			}
		}

		group.Set(tmp[i].CypherPublicKey, r[i].CypherPublicKey)
		group.Set(tmp[i].Z, r[i].Z)
	}

	return tmp
}

func GenerateIOMessages(nodeCount int, batchSize uint64,
	rounds []*globals.Round) ([]realtime.Slot, []realtime.Slot) {
	inputMsgs := make([]realtime.Slot, batchSize)
	outputMsgs := make([]realtime.Slot, batchSize)
	for i := uint64(0); i < batchSize; i++ {
		inputMsgs[i] = realtime.Slot{
			Slot:           i,
			CurrentID:      id.NewUserFromUint(i+1, nil),
			Message:        group.NewInt((42 + int64(i)) % 107), // Meaning of Life
			AssociatedData: group.NewInt((1 + int64(i)) % 107),
			CurrentKey:     group.NewInt(1),
		}
		outputMsgs[i] = realtime.Slot{
			Slot:      i,
			CurrentID: id.NewUserFromUint((i+1)%107, nil),
			Message:   group.NewInt((42 + int64(i)) % 107), // Meaning of Life
		}
	}
	for i := 0; i < nodeCount; i++ {
		// Now apply  permutations list to outputMsgs
		newOutMsgs := make([]realtime.Slot, batchSize)
		for j := uint64(0); j < batchSize; j++ {
			newOutMsgs[rounds[i].Permutations[j]] = outputMsgs[j]
		}

		copy(outputMsgs, newOutMsgs)
		outputMsgs = newOutMsgs
	}

	return inputMsgs, outputMsgs
}
*/

// Template function for running precomputation
func PrecompIterations(nodeCount int, batchSize uint64, iterations int) {
	/*	prime := large.NewInt(0)
		prime.SetString(PRIME, 16)

		grp := cyclic.NewGroup(prime, large.NewInt(5), large.NewInt(4))
		rounds := GenerateRounds(nodeCount, batchSize, grp)

		for i := 0; i < iterations; i++ {
			MultiNodePrecomp(nodeCount, batchSize, grp, rounds)
		}
	*/
}

// Run realtime simulation for given number of of iterations
func RealtimeIterations(nodeCount int, batchSize uint64, iterations int) {
	/*	prime := large.NewInt(0)
		prime.SetString(PRIME, 16)

		grp := cyclic.NewGroup(prime, large.NewInt(5), large.NewInt(4))

		rounds := GenerateRounds(nodeCount, batchSize, grp)

		// Rewrite permutation pattern
		for i := 0; i < nodeCount; i++ {
			for j := uint64(0); j < batchSize; j++ {
				// Shift by 1
				newj := (j + 1) % batchSize
				rounds[i].Permutations[j] = newj
			}
		}

		MultiNodePrecomp(nodeCount, batchSize, grp, rounds)

		for i := 0; i < iterations; i++ {
			tmpRounds := CopyRounds(nodeCount, rounds, grp)
			inputMsgs, outputMsgs := GenerateIOMessages(nodeCount, batchSize, tmpRounds)

			MultiNodeRealtime(nodeCount, batchSize, grp, tmpRounds, inputMsgs,
				outputMsgs)
		}
	*/
}
