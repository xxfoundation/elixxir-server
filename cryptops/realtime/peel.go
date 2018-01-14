package realtime

import (
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/server/server"
	"gitlab.com/privategrity/server/services"
)

type RealTimePeel struct{}

// DispatchBuilder to perform the peel operation. This needs to
// grab the aggregate R, S, and T inverses so they can be multipied against
// each message.
func (self RealTimePeel) Build(group *cyclic.Group,
	face interface{}) *services.DispatchBuilder {
	round := face.(*server.Round)
	batchSize := round.BatchSize
	outMessages := make([]*services.Message, batchSize)
	peelMessageKeys := make([][]*cyclic.Int, batchSize)

	for i, _ := range outMessages {
		outMessages[i] = services.NewMessage(uint64(i), 4, nil)
		// NOTE: This seems wrong but I'm not sure how we fix it. FIXME when we link
		//       everything up.
		peelMessageKeys[i] = []*cyclic.Int{
			round.R_INV[i], round.S_INV[i], round.T_INV[i]}
	}

	return &services.DispatchBuilder{
		BatchSize: batchSize,
		Saved: &peelMessageKeys,
		OutMessage: &outMessages,
		G: group}
}

func (self RealTimePeel) Run(g *cyclic.Group, in, out *services.Message,
	saved *[]*cyclic.Int) *services.Message {
	R_Inv, S_Inv, T_Inv := (*saved)[0], (*saved)[1], (*saved)[2]
	EncryptedMessage := in.Data[0]
	DecryptedMessage := out.Data[0]

	Peel(g, EncryptedMessage, DecryptedMessage, R_Inv, S_Inv, T_Inv)

	return out
}

// Peel (run only on the last node) multiplies the product of a
// sequence of all inverse R, S, and T keys from all nodes in order to
// remove all R, S, and T encryptions. Note that Peel should only be run
// on the final node in a cMix cluster.
func Peel(g *cyclic.Group, EncryptedMessage, DecryptedMessage,
	R_Inv, S_Inv, T_Inv *cyclic.Int) {
	g.Mul(EncryptedMessage, R_Inv, DecryptedMessage)
	g.Mul(DecryptedMessage, S_Inv, DecryptedMessage)
	g.Mul(DecryptedMessage, T_Inv, DecryptedMessage)
}
