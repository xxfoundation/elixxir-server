package cryptops

import (
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/server/server"
	"gitlab.com/privategrity/server/services"
)

/*//PERMUTE PHASE////////////////////////////////////////////////////////////*/

type PrecompPermute struct{}

func (perm PrecompPermute) Build(g *cyclic.Group, face interface{}) *services.DispatchBuilder {

	//get round from the empty interface
	round := face.(*server.Round)

	//Allocate Memory for output
	om := make([]*services.Message, round.BatchSize)

	/*CRYPTOGRAPHIC OPERATION BEGIN*/
	precompPermuteBuildCrypt(round, &om)
	/*CRYPTOGRAPHIC OPERATION END*/

	var sav [][]*cyclic.Int

	//Link the keys for randomization
	for i := uint64(0); i < round.BatchSize; i++ {
		roundSlc := []*cyclic.Int{
			server.G, round.S_INV[i], round.V_INV[i], round.Y_S[i], round.Y_V[i], round.G,
		}
		sav = append(sav, roundSlc)
	}

	db := services.DispatchBuilder{BatchSize: round.BatchSize, Saved: &sav, OutMessage: &om, G: g}

	return &db

}

//Implements cryptographic component of build
func precompPermuteBuildCrypt(round *server.Round, om *[]*services.Message) {

	for i := uint64(0); i < round.BatchSize; i++ {
		(*om)[i] = services.NewMessage(25, 4, nil)
	}

}

func (perm PrecompPermute) Run(grp *cyclic.Group, in, out *services.Message, saved *[]*cyclic.Int) *services.Message {
	G, S_INV, V_INV, Y_S, Y_V :=
		(*saved)[0], (*saved)[1], (*saved)[2], (*saved)[3], (*saved)[4]
	GCK := (*saved)[5]

	Message, Recipient, MessCyph, RecpCyph :=
		in.Data[0], in.Data[1], in.Data[2], in.Data[3]

	runCrypto(grp, out, G, S_INV, V_INV, Y_S, Y_V, GCK, Message, Recipient, MessCyph, RecpCyph)

	return out
}

func runCrypto(grp *cyclic.Group, out *services.Message, G, S_INV, V_INV, Y_S, Y_V, GCK, Message, Recipient, MessCyph, RecpCyph *cyclic.Int) {
	grp.Exp(G, Y_S, out.Data[0])
	grp.Mul(S_INV, out.Data[0], out.Data[0])
	grp.Mul(Message, out.Data[0], out.Data[0])

	grp.Exp(G, Y_V, out.Data[1])
	grp.Mul(V_INV, out.Data[1], out.Data[1])
	grp.Mul(Recipient, out.Data[1], out.Data[1])

	grp.Exp(GCK, Y_S, out.Data[2])
	grp.Mul(MessCyph, out.Data[2], out.Data[2])

	grp.Exp(GCK, Y_V, out.Data[3])
	grp.Mul(RecpCyph, out.Data[3], out.Data[3])
}
