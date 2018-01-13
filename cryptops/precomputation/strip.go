// Implements the Precomputation Strip phase

package precomputation

import (
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/server/server"
	"gitlab.com/privategrity/server/services"
)

type PrecompStrip struct{}

func (gen PrecompStrip) Build(g *cyclic.Group, face interface{}) *services.DispatchBuilder {

	//get round from the empty interface
	round := face.(*server.Round)

	//Allocate Memory for output
	om := make([]*services.Message, round.BatchSize)

	for i := uint64(0); i < round.BatchSize; i++ {
		om[i] = services.NewMessage(i, 3, nil)
	}

	var sav [][]*cyclic.Int

	//Link the keys for stripping
	for i := uint64(0); i < round.BatchSize; i++ {
		roundSlc := []*cyclic.Int{
			round.PI_Z,
		}
		sav = append(sav, roundSlc)
	}

	db := services.DispatchBuilder{BatchSize: round.BatchSize, Saved: &sav, OutMessage: &om, G: g}

	return &db

}

func (gen PrecompStrip) Run(g *cyclic.Group, in, out *services.Message, saved *[]*cyclic.Int) *services.Message {

	// Obtain PI_Z
	PI_Z := (*saved)[0]

	// Obtain message cypher text and encrypted message key
	messageCypherText, encryptedMessageKey := in.Data[0], in.Data[1]
	// Obtain recipient cypher text and encrypted recipient key
	recipientCypherText, encryptedRecipientKey := in.Data[2], in.Data[3]

	// Set output vars for the Message and Recipient Keys
	// NOTE: Out index 2 used for temporary computation
	messageKey, recipientKey, tmp := out.Data[0], out.Data[1], out.Data[2]

	// Separate operations into helper function for testing
	stripRunHelper(g, PI_Z, messageCypherText, encryptedMessageKey,
		recipientCypherText, encryptedRecipientKey, messageKey, recipientKey, tmp)

	return out

}

func stripRunHelper(g *cyclic.Group, PI_Z, messageCypherText, encryptedMessageKey, recipientCypherText,
	encryptedRecipientKey, messageKey, recipientKey, tmp *cyclic.Int) {
	// Helper function for Precomp Strip Run

	//Cancel PI_Z out of message cypher text to reveal global message private key
	g.Root(messageCypherText, PI_Z, tmp)

	// Invert the global message private key
	g.Inverse(tmp, tmp)

	// Use the inverted global message private key to remove the homomorphic encryption
	// from encrypted message key and reveal the message key
	g.Mul(tmp, encryptedMessageKey, messageKey)

	// Cancel PI_Z out of recipient cypher text to reveal global recipient private key
	g.Root(recipientCypherText, PI_Z, tmp)

	// Invert the global recipient private key
	g.Inverse(tmp, tmp)

	// Use the inverted global recipient private key to remove the homomorphic encryption
	// from encrypted recipient key and reveal the recipient key
	g.Mul(tmp, encryptedRecipientKey, recipientKey)

}
