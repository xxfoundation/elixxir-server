package realtime

import (
	"gitlab.com/privategrity/crypto/cyclic"
)

// Identify (run only on the last node) multiplies the product of a
// sequence of all inverse U and inverse V from all nodes in order to
// remove all V and U encryptions. Note that identify should only be run
// on the final node in a cMix cluster.
func Identify(g *cyclic.Group, EncryptedRecipient, DecryptedRecipient,
	U_Inv, V_Inv *cyclic.Int) {
	g.Mul(EncryptedRecipient, U_Inv, DecryptedRecipient)
	g.Mul(EncryptedRecipient, V_Inv, DecryptedRecipient)
}
