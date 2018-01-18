package realtime

import (
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/server/node"
	"gitlab.com/privategrity/server/services"
)

type RealTimePeel struct{}

// DispatchBuilder to perform the peel operation. This needs to
// grab the aggregate R, S, and T inverses so they can be multipied against
// each message.
func (self RealTimePeel) Build(group *cyclic.Group,
	face interface{}) *services.DispatchBuilder {
	round := face.(*node.Round)
	batchSize := round.BatchSize
	outMessages := make([]*services.Message, batchSize)
	peelMessageKeys := make([][]*cyclic.Int, batchSize)

	for i, _ := range outMessages {
		outMessages[i] = services.NewMessage(uint64(i), 4, nil)
		// NOTE: This seems wrong but I'm not sure how we fix it. FIXME when we link
		//       everything up.
		peelMessageKeys[i] = []*cyclic.Int{
			round.LastNode.MessagePrecomputation[i]}
	}

	return &services.DispatchBuilder{
		BatchSize:  batchSize,
		Saved:      &peelMessageKeys,
		OutMessage: &outMessages,
		G:          group}
}

func (self RealTimePeel) Run(g *cyclic.Group, in, out *services.Message,
	saved *[]*cyclic.Int) *services.Message {
	MessagePrecomputation := (*saved)[0]
	EncryptedMessage := in.Data[0]
	DecryptedMessage := out.Data[0]

	Peel(g, EncryptedMessage, DecryptedMessage, MessagePrecomputation)

	return out
}

// Peel (run only on the last node) multiplies the product of a
// sequence of all inverse R, S, and T keys from all nodes in order to
// remove all R, S, and T encryptions. Note that Peel should only be run
// on the final node in a cMix cluster.
func Peel(g *cyclic.Group, EncryptedMessage, DecryptedMessage,
	MessagePrecomputation *cyclic.Int) {
	g.Mul(EncryptedMessage, MessagePrecomputation, DecryptedMessage)
}
