package cryptops

import (
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/server/server"
	"gitlab.com/privategrity/server/services"
)

// Implements the Generation phase, which generates random keys for R, S, T, U,
// V, Y_R, Y_S, Y_T, Y_U, Y_V, and Z
type PrecompGeneration struct{}

func (gen PrecompGeneration) Build(g *cyclic.Group, face interface{}) *services.DispatchBuilder {

	//get round from the empty interface
	round := face.(*server.Round)

	/*CRYPTOGRAPHIC OPERATION BEGIN*/
	precompGenBuildCrypt(g, round)
	/*CRYPTOGRAPHIC OPERATION END*/

	//Allocate Memory for output
	om := make([]*services.Message, round.BatchSize)

	for i := uint64(0); i < round.BatchSize; i++ {
		om[i] = services.NewMessage(round.Permutations[i], 1, nil)
	}

	var sav [][]*cyclic.Int

	//Link the keys for randomization
	for i := uint64(0); i < round.BatchSize; i++ {
		roundSlc := []*cyclic.Int{
			round.R[i], round.S[i], round.T[i], round.U[i], round.V[i],
			round.Y_R[i], round.Y_S[i], round.Y_T[i], round.Y_U[i], round.Y_V[i],
		}
		sav = append(sav, roundSlc)
	}

	db := services.DispatchBuilder{BatchSize: round.BatchSize, Saved: &sav, OutMessage: &om, G: g}

	return &db

}

func (gen PrecompGeneration) Run(g *cyclic.Group, in, out *services.Message, saved *[]*cyclic.Int) *services.Message {
	//generate random values for all keys

	for i := uint64(0); i < uint64(len(*saved)); i++ {
		g.Gen((*saved)[i])
	}

	return out

}

//Implements crytoraphic component of build
func precompGenBuildCrypt(g *cyclic.Group, round *server.Round) {
	//Make the Permutation
	cyclic.Shuffle(&round.Permutations)

	//Generate the Global Cypher Key
	g.Gen(round.Z)
}
