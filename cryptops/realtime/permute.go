package realtime

import (
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/server/node"
	"gitlab.com/privategrity/server/services"
)

/*//PERMUTE PHASE////////////////////////////////////////////////////////////*/

type RealPermute struct{}

func (perm RealPermute) Build(g *cyclic.Group, face interface{}) *services.DispatchBuilder {

	//get round from the empty interface
	round := face.(*node.Round)

	//Allocate Memory for output
	om := make([]*services.Message, round.BatchSize)

	/*CRYPTOGRAPHIC OPERATION BEGIN*/
	realPermuteBuildCrypt(round, &om)
	/*CRYPTOGRAPHIC OPERATION END*/

	var sav [][]*cyclic.Int

	//Link the keys for randomization
	for i := uint64(0); i < round.BatchSize; i++ {
		roundSlc := []*cyclic.Int{
			round.S[i], round.V[i],
		}
		sav = append(sav, roundSlc)
	}

	db := services.DispatchBuilder{BatchSize: round.BatchSize, Saved: &sav, OutMessage: &om, G: g}

	return &db

}

//Implements cryptographic component of build
func realPermuteBuildCrypt(round *node.Round, om *[]*services.Message) {

	for i := uint64(0); i < round.BatchSize; i++ {
		(*om)[i] = services.NewMessage(round.Permutations[i], 2, nil)
	}

}

func (perm RealPermute) Run(grp *cyclic.Group, in, out *services.Message, saved *[]*cyclic.Int) *services.Message {
	S, V := (*saved)[0], (*saved)[1]

	Message, Recipient := in.Data[0], in.Data[1]

	runCrypto(grp, out, S, V, Message, Recipient)

	return out
}

func runCrypto(grp *cyclic.Group, out *services.Message, S, V, Message, Recipient *cyclic.Int) {
	grp.Mul(Message, S, out.Data[0])
	grp.Mul(Recipient, V, out.Data[1])
}
