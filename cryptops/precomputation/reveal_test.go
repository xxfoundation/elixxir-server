package precomputation

import (
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/server/node"
	"gitlab.com/privategrity/server/services"
	"testing"
)

func TestPrecomputationReveal(t *testing.T) {

	test := 6
	pass := 0

	bs := uint64(3)

	round := node.NewRound(bs)

	var im []services.Slot

	rng := cyclic.NewRandom(cyclic.NewInt(0), cyclic.NewInt(1000))

	grp := cyclic.NewGroup(cyclic.NewInt(101), cyclic.NewInt(23),
		cyclic.NewInt(29), rng)

	im = append(im, &SlotReveal{
		Slot:                 uint64(0),
		PartialMessageCypherText:     cyclic.NewInt(int64(39)),
		PartialRecipientCypherText: cyclic.NewInt(int64(13))})

	im = append(im, &SlotReveal{
		Slot:                 uint64(1),
		PartialMessageCypherText:     cyclic.NewInt(int64(86)),
		PartialRecipientCypherText: cyclic.NewInt(int64(87))})

	im = append(im, &SlotReveal{
		Slot:                 uint64(2),
		PartialMessageCypherText:     cyclic.NewInt(int64(39)),
		PartialRecipientCypherText: cyclic.NewInt(int64(51))})

	round.Z = cyclic.NewInt(53)

	results := [][]*cyclic.Int{
		{cyclic.NewInt(39), cyclic.NewInt(20)},
		{cyclic.NewInt(53), cyclic.NewInt(87)},
		{cyclic.NewInt(39), cyclic.NewInt(18)},
	}

	dc := services.DispatchCryptop(&grp, Reveal{}, nil, nil, round)

	for i := uint64(0); i < bs; i++ {
		dc.InChannel <- &im[i]
		trn := <-dc.OutChannel

		rtn := (*trn).(*SlotReveal)
		result := results[i]

		if result[0].Cmp(rtn.PartialMessageCypherText) != 0 ||
			result[1].Cmp(rtn.PartialRecipientCypherText) != 0 {
			t.Errorf("Test of PrecompReveal's cryptop failed on index: %v"+
				" Expected: %v,%v Received: %v,%v ", i,
				result[0].Text(10), result[1].Text(10),
				rtn.PartialMessageCypherText.Text(10),
				rtn.PartialRecipientCypherText.Text(10))
		} else {
			pass++
		}
	}

	println("PrecompReveal", pass, "out of", test, "tests passed.")

}

func TestPrecomputationRevealRun(t *testing.T) {
	bs := uint64(3)

	var im []*SlotReveal
	var om []*SlotReveal

	rng := cyclic.NewRandom(cyclic.NewInt(0), cyclic.NewInt(1000))

	grp := cyclic.NewGroup(cyclic.NewInt(101), cyclic.NewInt(23),
		cyclic.NewInt(29), rng)

	im = append(im, &SlotReveal{
		Slot:                 uint64(0),
		PartialMessageCypherText:     cyclic.NewInt(int64(39)),
		PartialRecipientCypherText: cyclic.NewInt(int64(13))})

	im = append(im, &SlotReveal{
		Slot:                 uint64(1),
		PartialMessageCypherText:     cyclic.NewInt(int64(86)),
		PartialRecipientCypherText: cyclic.NewInt(int64(87))})

	im = append(im, &SlotReveal{
		Slot:                 uint64(2),
		PartialMessageCypherText:     cyclic.NewInt(int64(39)),
		PartialRecipientCypherText: cyclic.NewInt(int64(51))})

	om = append(om, &SlotReveal{
		Slot:                 uint64(1),
		PartialMessageCypherText:     cyclic.NewInt(int64(0)),
		PartialRecipientCypherText: cyclic.NewInt(int64(0))})

	om = append(om, &SlotReveal{
		Slot:                 uint64(2),
		PartialMessageCypherText:     cyclic.NewInt(int64(0)),
		PartialRecipientCypherText: cyclic.NewInt(int64(0))})

	om = append(om, &SlotReveal{
		Slot:                 uint64(0),
		PartialMessageCypherText:     cyclic.NewInt(int64(0)),
		PartialRecipientCypherText: cyclic.NewInt(int64(0))})

	key := KeysReveal{
			Z: cyclic.NewInt(53),
		}

	results := [][]*cyclic.Int{
		{cyclic.NewInt(39), cyclic.NewInt(20)},
		{cyclic.NewInt(53), cyclic.NewInt(87)},
		{cyclic.NewInt(39), cyclic.NewInt(18)},
	}

	reveal := Reveal{}

	for i := uint64(0); i < bs; i++ {
		reveal.Run(&grp, im[i], om[i], &key)
	}

	for i := uint64(0); i < bs; i++ {
		if results[i][0].Cmp(om[i].PartialMessageCypherText) != 0 ||
			results[i][1].Cmp(om[i].PartialRecipientCypherText) != 0 {
			t.Errorf("%v - Expected: %v,%v Got: %v,%v",
				results[i][0].Text(10), results[i][1].Text(10),
				om[i].PartialMessageCypherText.Text(10),
				om[i].PartialRecipientCypherText.Text(10))
		}
	}
}
