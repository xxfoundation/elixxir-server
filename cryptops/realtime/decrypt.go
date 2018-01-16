// Implements the Realtime Decrypt phase

package realtime

import (
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/server/node"
	"gitlab.com/privategrity/server/services"
)

type RealTimeDecrypt struct{}

func (gen RealTimeDecrypt) Build(g *cyclic.Group, face interface{}) *services.DispatchBuilder {

	//get round from the empty interface
	round := face.(*node.Round)

	//Allocate Memory for output
	om := make([]*services.Message, round.BatchSize)

	for i := uint64(0); i < round.BatchSize; i++ {
		om[i] = services.NewMessage(i, 2, nil)
	}

	var sav [][]*cyclic.Int

	for i := uint64(0); i < round.BatchSize; i++ {
		roundSlc := []*cyclic.Int{
			round.R[i], round.U[i],
		}
		sav = append(sav, roundSlc)
	}

	db := services.DispatchBuilder{BatchSize: round.BatchSize, Saved: &sav, OutMessage: &om, G: g}

	return &db

}

func (gen RealTimeDecrypt) Run(g *cyclic.Group, in, out *services.Message, saved *[]*cyclic.Int) *services.Message {
	// Removes the encryption added by the Client while
	// simultaneously encrypting the message with unpermuted internode keys.

	// Obtain R and U
	R, U := (*saved)[0], (*saved)[1]

	// Obtain encrypted message, encrypted recipient ID, and client key (F) as input values
	encryptedMessageIn, encryptedRecipientIdIn, clientKey := in.Data[0], in.Data[1], in.Data[2]

	// Set output vars for the encrypted message and encrypted recipient ID
	// NOTE: Out index 1 used for temporary computation
	encryptedMessageOut, encryptedRecipientIdOut, tmp := out.Data[0], out.Data[1], out.Data[1]

	// Separate operations into helper function for testing
	decryptRunHelper(g, R, U, encryptedMessageIn, encryptedRecipientIdIn, clientKey,
		encryptedMessageOut, encryptedRecipientIdOut, tmp)

	return out

}

func decryptRunHelper(g *cyclic.Group, R, U, encryptedMessageIn, encryptedRecipientIdIn,
	clientKey, encryptedMessageOut, encryptedRecipientIdOut, tmp *cyclic.Int) {
	// Helper function for Realtime Decrypt Run

	// tmp = F * R
	g.Mul(clientKey, R, tmp)

	// EncryptedMessage * tmp
	g.Mul(encryptedMessageIn, tmp, encryptedMessageOut)

	// tmp = F * U
	g.Mul(clientKey, U, tmp)

	// EncryptedRecipientId * tmp
	g.Mul(encryptedRecipientIdIn, tmp, encryptedRecipientIdOut)

}
