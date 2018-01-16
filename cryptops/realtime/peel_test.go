package realtime

import (
	"gitlab.com/privategrity/crypto/cyclic"

	"testing"
)

// Smoke test test the peel function
func TestPeel(t *testing.T) {
	MessagePrecomp := cyclic.NewInt(41)

	grp := cyclic.NewGroup(cyclic.NewInt(43), cyclic.NewInt(5), cyclic.NewInt(23),
		cyclic.NewRandom(cyclic.NewInt(1), cyclic.NewInt(42)))

	EncryptedMessage := cyclic.NewInt(42)
	DecryptedMessage := cyclic.NewInt(0)

	ExpectedOutput := cyclic.NewInt(2) // 41*42 mod 43 => 2

	Peel(&grp, EncryptedMessage, DecryptedMessage, MessagePrecomp)

	if DecryptedMessage.Cmp(ExpectedOutput) != 0 {
		t.Errorf("Expected: %v, Got: %v", ExpectedOutput.Text(10),
			DecryptedMessage.Text(10))
	}
}
