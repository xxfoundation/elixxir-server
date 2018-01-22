package precomputation

import (
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/server/node"
	"gitlab.com/privategrity/server/services"
)

type PrecompEncrypt struct{}

// Build (PrecompEncrypt) allocates memory and links keys for the Encrypt phase
// of precomputation.
func (gen PrecompEncrypt) Build(g *cyclic.Group, face interface{}) *services.DispatchBuilder {

	//get round from the empty interface
	round := face.(*node.Round)

	//Allocate Memory for output
	om := make([]*services.Message, round.BatchSize)

	for i := uint64(0); i < round.BatchSize; i++ {
		om[i] = services.NewMessage(i, 2, nil)
	}

	var sav [][]*cyclic.Int

	//Link the keys for encryption
	for i := uint64(0); i < round.BatchSize; i++ {
		roundSlc := []*cyclic.Int{
			round.T_INV[i], round.Y_T[i], node.Grp.G, round.CypherPublicKey,
		}
		sav = append(sav, roundSlc)
	}

	db := services.DispatchBuilder{BatchSize: round.BatchSize, Saved: &sav, OutMessage: &om, G: g}

	return &db

}

// Run (PrecompEncrypt) executes the Encrypt phase of precomputation as
// detailed in the cMix technical document.

// Output of the Permute Phase is passed to the first Node which multiplies
// in its "Encrypted second unpermuted message keys" and the associated
// private keys into the Partial Message Cypher Text.
func (gen PrecompEncrypt) Run(g *cyclic.Group, in, out *services.Message,
	saved *[]*cyclic.Int) *services.Message {

	// Obtain T^-1, Y_T, and g
	T_INV, Y_T, serverG, globalCypherKey := (*saved)[0], (*saved)[1], (*saved)[2], (*saved)[3]

	// Obtain input values
	msgInput, cypherInput := in.Data[0], in.Data[1]

	// Set output vars for the encrypted message key and message cypher text
	// NOTE: Out index 1 used for temporary computation
	encryptedMessageKey, messageCypherText, tmp := out.Data[0], out.Data[1], out.Data[1]

	// Separate operations into helper function for testing
	encryptRunHelper(g, T_INV, Y_T, serverG, globalCypherKey, msgInput,
		cypherInput, encryptedMessageKey, messageCypherText, tmp)

	return out

}

// Helper function for PrecompEncrypt Run
func encryptRunHelper(g *cyclic.Group, T_INV, Y_T, serverG,
	globalCypherKey, msgInput, cypherInput,
	encryptedMessageKey, messageCypherText, tmp *cyclic.Int) {

	// Calculate g^(Y_T) into temp index of out.Data
	g.Exp(serverG, Y_T, tmp)

	// Calculate T^-1 * g^(Y_T) into temp index of out.Data
	g.Mul(T_INV, tmp, tmp)

	// Multiply message output of permute phase or previous encrypt phase
	// in msgInput with encrypted second unpermuted message keys into msgOutput
	g.Mul(msgInput, tmp, encryptedMessageKey)

	// Calculate g^((piZ) * Y_T) into temp index of out.Data
	// roundG = g^(piZ), so raise roundG to Y_T
	g.Exp(globalCypherKey, Y_T, tmp)

	// Multiply cypher text output of permute phase or previous encrypt phase
	// in cypherInput with encrypted second unpermuted message key private key into cypherOutput
	g.Mul(cypherInput, tmp, messageCypherText)

}
