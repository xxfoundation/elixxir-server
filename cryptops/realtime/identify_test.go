package realtime

import (
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/server/globals"
	"gitlab.com/privategrity/server/services"
	"testing"
)

func TestRealTimeIdentify(t *testing.T) {
	var im []services.Slot
	batchSize := uint64(2)
	round := globals.NewRound(batchSize)
	round.RecipientPrecomputation = make([]*cyclic.Int, batchSize)
	round.RecipientPrecomputation[0] = cyclic.NewInt(42)
	round.RecipientPrecomputation[1] = cyclic.NewInt(42)

	grp := cyclic.NewGroup(cyclic.NewInt(43), cyclic.NewInt(5), cyclic.NewInt(23),
		cyclic.NewRandom(cyclic.NewInt(1), cyclic.NewInt(42)))

	im = append(im, &SlotIdentify{
		Slot:                 0,
		EncryptedRecipientID: cyclic.NewInt(42)})

	im = append(im, &SlotIdentify{
		Slot:                 1,
		EncryptedRecipientID: cyclic.NewInt(1)})

	ExpectedOutputs := []*cyclic.Int{cyclic.NewInt(1), cyclic.NewInt(42)}

	dc := services.DispatchCryptop(&grp, Identify{}, nil, nil, round)

	for i := uint64(0); i < batchSize; i++ {
		dc.InChannel <- &im[i]
		trn := <-dc.OutChannel

		rtn := (*trn).(*SlotIdentify)
		ExpectedOutput := ExpectedOutputs[i]

		if rtn.EncryptedRecipientID.Cmp(ExpectedOutput) != 0 {
			t.Errorf("%v - Expected: %v, Got: %v", i, ExpectedOutput.Text(10),
				rtn.EncryptedRecipientID.Text(10))
		}
	}
}

// Smoke test test the identify function
func TestIdentifyRun(t *testing.T) {
	keys := KeysIdentify{
		RecipientPrecomputation: cyclic.NewInt(42)}

	grp := cyclic.NewGroup(cyclic.NewInt(43), cyclic.NewInt(5), cyclic.NewInt(23),
		cyclic.NewRandom(cyclic.NewInt(1), cyclic.NewInt(42)))

	// EncryptedRecipient := cyclic.NewInt(42)
	// DecryptedRecipient := cyclic.NewInt(0)

	im := SlotIdentify{
		Slot:                 0,
		EncryptedRecipientID: cyclic.NewInt(42)}

	om := SlotIdentify{
		Slot:                 0,
		EncryptedRecipientID: cyclic.NewInt(0)}

	ExpectedOutput := cyclic.NewInt(1) // 42*42 mod 43 => 1

	// Identify(&grp, EncryptedRecipient, DecryptedRecipient, RecipientPrecomp)
	identify := Identify{}
	identify.Run(&grp, &im, &om, &keys)

	if om.EncryptedRecipientID.Cmp(ExpectedOutput) != 0 {
		t.Errorf("Expected: %v, Got: %v", ExpectedOutput.Text(10),
			om.EncryptedRecipientID.Text(10))
	}
}
