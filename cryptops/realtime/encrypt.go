package realtime

import (
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/server/node"
	"gitlab.com/privategrity/server/services"
)

// Encrypt phase of realtime operations
type RealtimeEncrypt struct{}

func (self RealtimeEncrypt) Build(g *cyclic.Group,
	face interface{}) *services.DispatchBuilder {

	round := face.(*node.Round)

	outMessages := make([]*services.Message, round.BatchSize)

	keyCache := make([][]*cyclic.Int, round.BatchSize)

	for i := uint64(0); i < round.BatchSize; i++ {
		outMessages[i] = services.NewMessage(i, 1, nil)

		keys := []*cyclic.Int{
			round.T[i], round.Z,
		}
		keyCache[i] = keys
	}

	db := services.DispatchBuilder{BatchSize: round.BatchSize,
		Saved: &keyCache, OutMessage: &outMessages, G: g}

	return &db
}

// Cryptographic operation
func encryptMessage(g *cyclic.Group, permutedCombinedMessageKeys,
	secondUnpermutedInternodeMessageKey, nodeCipherKey,
	encryptedMessage *cyclic.Int) {
	g.Mul(permutedCombinedMessageKeys,
		secondUnpermutedInternodeMessageKey, encryptedMessage)
	g.Mul(encryptedMessage, nodeCipherKey, encryptedMessage)
}

func (self RealtimeEncrypt) Run(g *cyclic.Group, in, out *services.Message,
	saved *[]*cyclic.Int) *services.Message {

	// M dot Pi R dot Pi S [w] in the docs for the first node
	permutedCombinedMessageKeys := (*in).Data[0]

	secondUnpermutedInternodeMessageKey := (*saved)[0]
	nodeCipherKey := (*saved)[1]

	encryptedMessage := out.Data[0]
	encryptMessage(g, permutedCombinedMessageKeys,
		secondUnpermutedInternodeMessageKey, nodeCipherKey,
		encryptedMessage)

	return out
}
