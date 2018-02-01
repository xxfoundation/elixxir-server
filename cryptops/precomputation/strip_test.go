package precomputation

import (
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/server/globals"
	"gitlab.com/privategrity/server/services"
	"testing"
)

func TestStrip(t *testing.T) {
	// NOTE: Does not test correctness.

	test := 3
	pass := 0

	batchSize := uint64(3)

	round := globals.NewRound(batchSize)

	rng := cyclic.NewRandom(cyclic.NewInt(2), cyclic.NewInt(2000))

	group := cyclic.NewGroup(cyclic.NewInt(199), cyclic.NewInt(11),
		cyclic.NewInt(13), rng)

	var inMessages []services.Slot

	inMessages = append(inMessages, &SlotStripIn{Slot: uint64(0),
		RoundMessagePrivateKey:   cyclic.NewInt(39),
		RoundRecipientPrivateKey: cyclic.NewInt(13)})

	inMessages = append(inMessages, &SlotStripIn{Slot: uint64(1),
		RoundMessagePrivateKey:   cyclic.NewInt(86),
		RoundRecipientPrivateKey: cyclic.NewInt(87)})

	inMessages = append(inMessages, &SlotStripIn{Slot: uint64(2),
		RoundMessagePrivateKey:   cyclic.NewInt(39),
		RoundRecipientPrivateKey: cyclic.NewInt(51)})

	globals.InitLastNode(round)
	round.LastNode.EncryptedMessagePrecomputation[0] = cyclic.NewInt(41)
	round.LastNode.EncryptedRecipientPrecomputation[0] = cyclic.NewInt(74)
	round.LastNode.EncryptedMessagePrecomputation[1] = cyclic.NewInt(8)
	round.LastNode.EncryptedRecipientPrecomputation[1] = cyclic.NewInt(49)
	round.LastNode.EncryptedMessagePrecomputation[2] = cyclic.NewInt(91)
	round.LastNode.EncryptedRecipientPrecomputation[2] = cyclic.NewInt(73)

	expected := []SlotStripOut{
		SlotStripOut{Slot: uint64(0),
			MessagePrecomputation:   cyclic.NewInt(10),
			RecipientPrecomputation: cyclic.NewInt(136)},
		SlotStripOut{Slot: uint64(1),
			MessagePrecomputation:   cyclic.NewInt(119),
			RecipientPrecomputation: cyclic.NewInt(7)},
		SlotStripOut{Slot: uint64(2),
			MessagePrecomputation:   cyclic.NewInt(10),
			RecipientPrecomputation: cyclic.NewInt(59)},
	}

	dc := services.DispatchCryptop(&group, Strip{}, nil, nil, round)

	for i := uint64(0); i < batchSize; i++ {
		dc.InChannel <- &(inMessages[i])
		act := <-dc.OutChannel
		actual := (*act).(*SlotStripOut)

		if actual.SlotID() != expected[i].SlotID() {
			t.Errorf("Test of Precomputation Strip's cryptop failed Slot"+
				"ID Test on index: %v; Expected: %v; Actual: %v\n", i,
				expected[i].SlotID(), actual.SlotID())
		} else if actual.MessagePrecomputation.Cmp(
			expected[i].MessagePrecomputation) != 0 {
			t.Errorf("Test of Precomputation Strip's cryptop failed"+
				" MessagePrecomputation "+
				"on index: %v; Expected: %v; Actual: %v\n", i,
				expected[i].MessagePrecomputation.Text(10),
				actual.MessagePrecomputation.Text(10))
		} else if actual.RecipientPrecomputation.Cmp(
			expected[i].RecipientPrecomputation) != 0 {
			t.Errorf("Test of Precomputation Strip's cryptop failed"+
				" RecipientPrecomputation "+
				"on index: %v; Expected: %v; Actual: %v\n", i,
				expected[i].RecipientPrecomputation.Text(10),
				actual.RecipientPrecomputation.Text(10))
		} else {
			pass++
		}
	}

	println("Precomputation Strip", pass, "out of", test, "tests "+
		"passed.")
}
