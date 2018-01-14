package realtime

import (
	"gitlab.com/privategrity/crypto/cyclic"
)

type Identify struct{}


// DispatchBuilder to perform the identify operation. This needs to
// grab the aggregate U and V inverses so they can be multipied against
// each message.
func (self Identify) Build(group *cyclic.Group,
	face interface{}) *services.DispatchBuilder {
	round := face.(*server.Round)
	batchSize := round.BatchSize
	outMessages := make([]*services.Message, batchSize)
	identifyMessageKeys := make([][]*cyclic.Int, batchSize)

	for i, outMessage := range outMessages {
		outMessage = services.NewMessage(i, 4, nil)
		// NOTE: This seems wrong but I'm not sure how we fix it. FIXME when we link
		//       everything up.
		identifyMessageKeys[i] = []*cyclic.Int{
			round.U_INV, round.V_INV}
	}

	return &services.DispatchBuilder{
		BatchSize: batchSize,
		Saved: identifyMessageKeys,
		OutMessage: &outMessages,
		G: group}
}

func (self Identify) Run(g *cyclic.Group, in, out *services.Message,
	saved *[]*cyclic.Int) *services.Message {
	U_Inv, V_Inv := (*saved)[0], (*saved)[1]
	EncryptedRecipient := in.Data[0]
	DecryptedRecipient := out.Data[0]
	Identify(g, EncryptedRecipient, DecryptedRecipient, U_Inv, V_Inv)
}

// Identify (run only on the last node) multiplies the product of a
// sequence of all inverse U and inverse V from all nodes in order to
// remove all V and U encryptions. Note that identify should only be run
// on the final node in a cMix cluster.
func Identify(g *cyclic.Group, EncryptedRecipient, DecryptedRecipient,
	U_Inv, V_Inv *cyclic.Int) {
	g.Mul(EncryptedRecipient, U_Inv, DecryptedRecipient)
	g.Mul(EncryptedRecipient, V_Inv, DecryptedRecipient)
}
