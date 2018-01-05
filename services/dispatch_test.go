package services

import (
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/server/server"
	"testing"
)

type testCryptop struct{}

func (cry testCryptop) run(g *cyclic.Group, in, out *Message, saved *[]*cyclic.Int) *Message {

	out.Data[0] = out.Data[0].Add(in.Data[0], (*saved)[0])

	return out
}

func (cry testCryptop) build(g *cyclic.Group, face interface{}) *DispatchBuilder {

	round := face.(*server.Round)

	om := make([]*Message, round.BatchSize)

	i := uint64(0)
	for i < round.BatchSize {
		om[i] = NewMessage(i, 1, nil)
		i++
	}

	var sav [][]*cyclic.Int

	i = uint64(0)
	for i < round.BatchSize {
		sav = append(sav, []*cyclic.Int{round.R[i]})
		i++
	}

	db := DispatchBuilder{BatchSize: round.BatchSize, Saved: &sav, OutMessage: &om, group: g}

	return &db
}

func TestDispatchCryptop(t *testing.T) {

	test := 4
	pass := 0

	bs := uint64(4)

	round := server.NewRound(bs)

	var im []*Message

	i := uint64(0)
	for i < bs {
		im = append(im, &Message{uint64(i), []*cyclic.Int{cyclic.NewInt(int64(i + 1))}})
		round.R[i] = cyclic.NewInt(int64(2 * (i + 1)))
		i++
	}

	result := []*cyclic.Int{
		cyclic.NewInt(3), cyclic.NewInt(6), cyclic.NewInt(9), cyclic.NewInt(12),
	}

	gen := cyclic.NewGen(cyclic.NewInt(0), cyclic.NewInt(1000))

	g := cyclic.NewGroup(cyclic.NewInt(11), cyclic.NewInt(5), gen)

	dc := DispatchCryptop(&g, testCryptop{}, nil, nil, round)

	i = 0
	for i < bs {

		dc.InChannel <- im[i]
		rtn := <-dc.OutChannel

		if rtn.Data[0].Cmp(result[i]) != 0 {
			t.Errorf("Test of Dispatcher failed at index: %v Expected: %v;",
				" Actual: %v", i, result[0].Text(10), rtn.Data[0].Text(10))
		} else {
			pass++
		}

		i++
	}

	println("Dispatcher", pass, "out of", test, "tests passed.")

}
