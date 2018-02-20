// Copyright © 2018 Privategrity Corporation
//
// All rights reserved.
package precomputation

import (
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/server/globals"
	"gitlab.com/privategrity/server/services"
	"testing"
)

// Not sure if the input data represents real data accurately
// Expected data was generated by the cryptop.
// Right now this tests for regression, not correctness.
func TestPrecompDecrypt(t *testing.T) {
	test := 3
	pass := 0
	batchSize := uint64(3)
	round := globals.NewRound(batchSize)
	rng := cyclic.NewRandom(cyclic.NewInt(0), cyclic.NewInt(1000))
	group := cyclic.NewGroup(cyclic.NewInt(17), cyclic.NewInt(5), cyclic.NewInt(7), rng)
	globals.Grp = &group

	round.CypherPublicKey = cyclic.NewInt(13)

	var im []services.Slot

	im = append(im, &SlotDecrypt{
		Slot:                         uint64(0),
		EncryptedMessageKeys:         cyclic.NewInt(12),
		EncryptedRecipientIDKeys:     cyclic.NewInt(7),
		PartialMessageCypherText:     cyclic.NewInt(3),
		PartialRecipientIDCypherText: cyclic.NewInt(8)})

	im = append(im, &SlotDecrypt{
		Slot:                         uint64(1),
		EncryptedMessageKeys:         cyclic.NewInt(2),
		EncryptedRecipientIDKeys:     cyclic.NewInt(4),
		PartialMessageCypherText:     cyclic.NewInt(9),
		PartialRecipientIDCypherText: cyclic.NewInt(16)})

	im = append(im, &SlotDecrypt{
		Slot:                         uint64(2),
		EncryptedMessageKeys:         cyclic.NewInt(14),
		EncryptedRecipientIDKeys:     cyclic.NewInt(99),
		PartialMessageCypherText:     cyclic.NewInt(96),
		PartialRecipientIDCypherText: cyclic.NewInt(5)})

	round.R_INV[0] = cyclic.NewInt(5)
	round.U_INV[0] = cyclic.NewInt(9)
	round.Y_R[0] = cyclic.NewInt(15)
	round.Y_U[0] = cyclic.NewInt(2)

	round.R_INV[1] = cyclic.NewInt(8)
	round.U_INV[1] = cyclic.NewInt(1)
	round.Y_R[1] = cyclic.NewInt(13)
	round.Y_U[1] = cyclic.NewInt(6)

	round.R_INV[2] = cyclic.NewInt(38)
	round.U_INV[2] = cyclic.NewInt(100)
	round.Y_R[2] = cyclic.NewInt(44)
	round.Y_U[2] = cyclic.NewInt(32)

	expected := [][]*cyclic.Int{{
		cyclic.NewInt(11), cyclic.NewInt(10),
		cyclic.NewInt(12), cyclic.NewInt(9),
	}, {
		cyclic.NewInt(11), cyclic.NewInt(2),
		cyclic.NewInt(15), cyclic.NewInt(1),
	}, {
		cyclic.NewInt(14), cyclic.NewInt(6),
		cyclic.NewInt(11), cyclic.NewInt(5),
	}}

	dispatch := services.DispatchCryptop(
		&group, Decrypt{}, nil, nil, round)

	for i := 0; i < len(im); i++ {

		dispatch.InChannel <- &(im[i])
		actual := <-dispatch.OutChannel

		act := (*actual).(*SlotDecrypt)

		expectedVal := expected[i]

		if act.EncryptedMessageKeys.Cmp(expectedVal[0]) != 0 {
			t.Errorf("Test of Precomputation Decrypt's cryptop failed Message"+
				"Keys Test on index: %v; Expected: %v, Actual: %v", i,
				expectedVal[0].Text(10), act.EncryptedMessageKeys.Text(10))
		} else if act.EncryptedRecipientIDKeys.Cmp(expectedVal[1]) != 0 {
			t.Errorf("Test of Precomputation Decrypt's cryptop failed Recipient"+
				"Keys Test on index: %v; Expected: %v, Actual: %v", i,
				expectedVal[1].Text(10), act.EncryptedRecipientIDKeys.Text(10))
		} else if act.PartialMessageCypherText.Cmp(expectedVal[2]) != 0 {
			t.Errorf("Test of Precomputation Decrypt's cryptop failed Message"+
				"Cypher Test on index: %v; Expected: %v, Actual: %v", i,
				expectedVal[2].Text(10), act.PartialMessageCypherText.Text(10))
		} else if act.PartialRecipientIDCypherText.Cmp(expectedVal[3]) != 0 {
			t.Errorf("Test of Precomputation Decrypt's cryptop failed Recipient"+
				"Cypher Test on index: %v; Expected: %v, Actual: %v", i,
				expectedVal[3].Text(10), act.PartialRecipientIDCypherText.Text(10))
		} else {
			pass++
		}
	}

	println("Precomputation Decrypt:", pass, "out of", test, "tests passed.")
}
