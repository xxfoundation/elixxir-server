package precomputation

import (
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/server/node"
	"gitlab.com/privategrity/server/services"
)

type PrecompShare struct{}

func (shar PrecompShare) Build(g *cyclic.Group, face interface{}) *services.DispatchBuilder {

	//get round from the empty interface
	round := face.(*node.Round)

	//Allocate Memory for output
	om := make([]*services.Message, round.BatchSize)

	for i := uint64(0); i < round.BatchSize; i++ {
		om[i] = services.NewMessage(i, 1, nil)
	}

	var sav [][]*cyclic.Int

	//Link the keys for encryption
	for i := uint64(0); i < round.BatchSize; i++ {
		roundSlc := []*cyclic.Int{
			round.Z,
		}
		sav = append(sav, roundSlc)
	}

	db := services.DispatchBuilder{BatchSize: round.BatchSize, Saved: &sav, OutMessage: &om, G: g}

	return &db

}

func (shar PrecompShare) Run(g *cyclic.Group, in, out *services.Message, saved *[]*cyclic.Int) *services.Message {

	// Obtain Z
	Z := (*saved)[0]

	// Obtain input values
	keyInput := in.Data[0]

	// Obtain output values
	keyOutput := out.Data[0]

	// Separate operations into helper function for testing
	shareRunHelper(g, Z, keyInput, keyOutput)

	return out

}

func shareRunHelper(g *cyclic.Group, Z, keyInput, keyOutput *cyclic.Int) {

	g.Exp(keyInput, Z, keyOutput)

}
