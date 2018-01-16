package precomputation

import (
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/server/server"
	"gitlab.com/privategrity/server/services"
)

// Decrypt phase: transform first unpermuted internode keys and partial cipher
// tests into the data that the permute phase needs
type PrecompDecrypt struct{}

// in.Data[0]: first unpermuted internode message key from previous node
// in.Data[1]: first unpermuted internode recipient ID key from previous node
// in.Data[2]: partial cipher test for first unpermuted internode message key
//             from previous node
// in.Data[3]: partial cipher test for first unpermuted internode recipient
//             ID key from previous node
// Each out datum corresponds to the in datum, with the required data from
// this node combined as specified Therefore, out must be of width 4
func (self PrecompDecrypt) Run(g *cyclic.Group, in, out *services.Message,
	saved *[]*cyclic.Int) *services.Message {

	R_INV := (*saved)[0]
	Y_R := (*saved)[1]
	U_INV := (*saved)[2]
	Y_U := (*saved)[3]
	globalCypherKey := (*saved)[4]
	globalHomomorphicGenerator := (*saved)[5]

	combineFirstUnpermutedInternodeKeys(g, in.Data[0], Y_R, R_INV,
		globalHomomorphicGenerator, out.Data[0])
	combineFirstUnpermutedInternodeKeys(g, in.Data[1], Y_U, U_INV,
		globalHomomorphicGenerator, out.Data[1])

	combinePartialCipherTests(g, in.Data[2], Y_R, globalCypherKey,
		out.Data[2])
	combinePartialCipherTests(g, in.Data[3], Y_U, globalCypherKey,
		out.Data[3])

	return out
}

// cryptographic function
func combineFirstUnpermutedInternodeKeys(
	g *cyclic.Group, firstUnpermutedInternodeKey, privateKey,
	publicKeyInverse, globalHomomorphicGenerator, result *cyclic.Int) {

	g.Exp(globalHomomorphicGenerator, privateKey, result)
	g.Mul(publicKeyInverse, result, result)
	g.Mul(firstUnpermutedInternodeKey, result, result)
}

// cryptographic function
func combinePartialCipherTests(
	g *cyclic.Group, partialCipherTest, privateKey, globalCypherKey,
	result *cyclic.Int) {

	g.Exp(globalCypherKey, privateKey, result)
	g.Mul(partialCipherTest, result, result)
}

func (self PrecompDecrypt) Build(group *cyclic.Group,
	face interface{}) *services.DispatchBuilder {
	round := face.(*server.Round)
	batchSize := round.BatchSize
	outMessage := make([]*services.Message, batchSize)
	var keysForMessages [][]*cyclic.Int

	for i := uint64(0); i < batchSize; i++ {
		outMessage[i] = services.NewMessage(i, 4, nil)

		keysForThisMessage := []*cyclic.Int{
			round.R_INV[i], round.Y_R[i], round.U_INV[i],
			round.Y_U[i], round.CypherPublicKey, server.Grp.G}

		keysForMessages = append(keysForMessages, keysForThisMessage)
	}

	return &services.DispatchBuilder{
		BatchSize: batchSize, Saved: &keysForMessages,
		OutMessage: &outMessage, G: group}
}
