package realtime

import (
	"gitlab.com/privategrity/crypto/cyclic"

	"testing"
)

// Smoke test test the identify function
func TestIdentify(t *testing.T) {
	RecipientPrecomp := cyclic.NewInt(42)

	grp := cyclic.NewGroup(cyclic.NewInt(43), cyclic.NewInt(5), cyclic.NewInt(23),
		cyclic.NewRandom(cyclic.NewInt(1), cyclic.NewInt(42)))

	EncryptedRecipient := cyclic.NewInt(42)
	DecryptedRecipient := cyclic.NewInt(0)

	ExpectedOutput := cyclic.NewInt(1) // 42*42 mod 43 => 1

	Identify(&grp, EncryptedRecipient, DecryptedRecipient, RecipientPrecomp)

	if DecryptedRecipient.Cmp(ExpectedOutput) != 0 {
		t.Errorf("Expected: %v, Got: %v", ExpectedOutput.Text(10),
			DecryptedRecipient.Text(10))
	}
}
