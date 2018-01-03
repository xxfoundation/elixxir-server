package services

import (
	"fmt"
	"gitlab.com/privategrity/crypto/cyclic"
	"testing"
)

type testCryptop struct{}

func (cry testCryptop) run(in, out *Message, saved *[]*cyclic.Int) *Message {

	out.Data[0] = out.Data[0].Add(in.Data[0], (*saved)[0])

	return out
}

func TestDispatchCryptop(t *testing.T) {

	test := 4
	pass := 0

	var im []*Message

	bs := uint64(4)

	i := uint64(0)
	for i < bs {
		im = append(im, &Message{uint64(i), []*cyclic.Int{cyclic.NewInt(int64(i + 1))}})
		i++
	}

	var om []*Message

	i = 0
	for i < bs {
		om = append(om, &Message{uint64(i), []*cyclic.Int{cyclic.NewInt(int64(0))}})
		i++
	}

	saved := &[][]*cyclic.Int{
		[]*cyclic.Int{cyclic.NewInt(2)}, []*cyclic.Int{cyclic.NewInt(4)},
		[]*cyclic.Int{cyclic.NewInt(6)}, []*cyclic.Int{cyclic.NewInt(8)},
	}

	result := []*cyclic.Int{
		cyclic.NewInt(3), cyclic.NewInt(6), cyclic.NewInt(9), cyclic.NewInt(12),
	}

	dc, _ := DispatchCryptop(testCryptop{}, bs, &om, saved, nil, nil)

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
