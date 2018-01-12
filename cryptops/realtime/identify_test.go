package realtime

import (
	"gitlab.com/privategrity/crypto/cyclic"

	"testing"
)


// Smoke test test the identify function
func TestIdentify(t *testing.T) {
	UInv := cyclic.NewInt(21)
	VInv := cyclic.NewInt(2)

	g := cyclic.NewGroup(cyclic.NewInt(43), cyclic.NewInt(5),
		cyclic.NewGen(cyclic.NewInt(1), cyclic.NewInt(42)))

	EncryptedRecipient := cyclic.NewInt(42)
	DecryptedRecipient := cyclic.NewInt(0)

	ExpectedOutput := cyclic.NewInt(41) // 21*2*42 mod 43 => 41

	Identify(&g, EncryptedRecipient, DecryptedRecipient, UInv, VInv)

	if DecryptedRecipient.Cmp(ExpectedOutput) != 0 {
		t.Errorf("Expected: %v, Got: %v", ExpectedOutput.Text(10),
			DecryptedRecipient.Text(10))
	}
}
