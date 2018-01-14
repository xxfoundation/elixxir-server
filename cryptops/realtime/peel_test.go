package realtime

import (
	"gitlab.com/privategrity/crypto/cyclic"

	"testing"
)


// Smoke test test the peel function
func TestPeel(t *testing.T) {
	RInv := cyclic.NewInt(21)
	SInv := cyclic.NewInt(2)
	TInv := cyclic.NewInt(2)

	g := cyclic.NewGroup(cyclic.NewInt(43), cyclic.NewInt(5),
		cyclic.NewGen(cyclic.NewInt(1), cyclic.NewInt(42)))

	EncryptedMessage := cyclic.NewInt(42)
	DecryptedMessage := cyclic.NewInt(0)

	ExpectedOutput := cyclic.NewInt(2) // 21*2*42*2 mod 43 => 2

	Peel(&g, EncryptedMessage, DecryptedMessage, RInv, SInv, TInv)

	if DecryptedMessage.Cmp(ExpectedOutput) != 0 {
		t.Errorf("Expected: %v, Got: %v", ExpectedOutput.Text(10),
			DecryptedMessage.Text(10))
	}
}
