package graphs

import (
	"fmt"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/large"
	"gitlab.com/elixxir/crypto/shuffle"
	"gitlab.com/elixxir/server/node"
	"gitlab.com/elixxir/server/services"
	"reflect"
	"runtime"
	"sync/atomic"
	"testing"
)

// tests that the PermuteSubStream struct meets the permuteSubStreamInterface
func TestPermuteSubStream_Interface(t *testing.T) {
	pss := &PermuteSubStream{}

	var face interface{}

	face = pss

	_, ok := face.(permuteSubStreamInterface)

	if !ok {
		t.Errorf("permuteSubStreamInterface: PermuteSubStream does not meet interface")
	}

}

//Tests that link properly connects the substream to a stream
func TestPermuteSubStream_Link(t *testing.T) {
	primeString := "FFFFFFFFFFFFFFFFC90FDAA22168C234C4C6628B80DC1CD1" +
		"29024E088A67CC74020BBEA63B139B22514A08798E3404DD" +
		"EF9519B3CD3A431B302B0A6DF25F14374FE1356D6D51C245" +
		"E485B576625E7EC6F44C42E9A637ED6B0BFF5CB6F406B7ED" +
		"EE386BFB5A899FA5AE9F24117C4B1FE649286651ECE45B3D" +
		"C2007CB8A163BF0598DA48361C55D39A69163FA8FD24CF5F" +
		"83655D23DCA3AD961C62F356208552BB9ED529077096966D" +
		"670C354E4ABC9804F1746C08CA18217C32905E462E36CE3B" +
		"E39E772C180E86039B2783A2EC07A28FB5C55DF06F4C52C9" +
		"DE2BCBF6955817183995497CEA956AE515D2261898FA0510" +
		"15728E5A8AACAA68FFFFFFFFFFFFFFFF"
	grp := cyclic.NewGroup(large.NewIntFromString(primeString, 16), large.NewInt(2), large.NewInt(1283))

	p := PermuteSubStream{}

	expandedBatchSize := uint32(6)

	ib1 := grp.NewIntBuffer(6, grp.NewInt(7))
	ib2 := grp.NewIntBuffer(6, grp.NewInt(12))

	cs1 := make([]*cyclic.Int, expandedBatchSize)
	cs2 := make([]*cyclic.Int, expandedBatchSize)

	for i := uint32(0); i < expandedBatchSize; i++ {
		cs1[i] = grp.NewInt(int64(i + 1))
		cs2[i] = grp.NewInt(int64(i + 1))
	}

	round := node.NewRound(grp, 4, expandedBatchSize)

	round.Permutations = []uint32{3, 5, 0, 1, 2, 4}

	p.LinkStreams(expandedBatchSize, round.Permutations, PermuteIO{ib1, cs1}, PermuteIO{ib2, cs2})

	if !reflect.DeepEqual(p.permutations, round.Permutations) {
		t.Errorf("PermuteStream.Link: Permutation not linked properly")
	}

	compareIntBuffers(grp, ib1, p.inputs[0], "PermuteStream.Link", t)
	compareIntBuffers(grp, ib2, p.inputs[1], "PermuteStream.Link", t)

	compareIntSlices(grp, cs1, p.outputs[0], "PermuteStream.Link", t)
	compareIntSlices(grp, cs2, p.outputs[1], "PermuteStream.Link", t)
}

//tests that getting the PermuteSubStream object from the PermuteSubStreamInterface works correctly
func TestPermuteSubStream_getSubStream(t *testing.T) {
	primeString := "FFFFFFFFFFFFFFFFC90FDAA22168C234C4C6628B80DC1CD1" +
		"29024E088A67CC74020BBEA63B139B22514A08798E3404DD" +
		"EF9519B3CD3A431B302B0A6DF25F14374FE1356D6D51C245" +
		"E485B576625E7EC6F44C42E9A637ED6B0BFF5CB6F406B7ED" +
		"EE386BFB5A899FA5AE9F24117C4B1FE649286651ECE45B3D" +
		"C2007CB8A163BF0598DA48361C55D39A69163FA8FD24CF5F" +
		"83655D23DCA3AD961C62F356208552BB9ED529077096966D" +
		"670C354E4ABC9804F1746C08CA18217C32905E462E36CE3B" +
		"E39E772C180E86039B2783A2EC07A28FB5C55DF06F4C52C9" +
		"DE2BCBF6955817183995497CEA956AE515D2261898FA0510" +
		"15728E5A8AACAA68FFFFFFFFFFFFFFFF"
	grp := cyclic.NewGroup(large.NewIntFromString(primeString, 16), large.NewInt(2), large.NewInt(1283))

	p := PermuteSubStream{}

	expandedBatchSize := uint32(6)

	ib1 := grp.NewIntBuffer(6, grp.NewInt(7))
	ib2 := grp.NewIntBuffer(6, grp.NewInt(12))

	cs1 := make([]*cyclic.Int, expandedBatchSize)
	cs2 := make([]*cyclic.Int, expandedBatchSize)

	for i := uint32(0); i < expandedBatchSize; i++ {
		cs1[i] = grp.NewInt(int64(i + 1))
		cs2[i] = grp.NewInt(int64(i + 1))
	}

	round := node.NewRound(grp, 4, expandedBatchSize)

	round.Permutations = []uint32{3, 5, 0, 1, 2, 4}

	p.LinkStreams(expandedBatchSize, round.Permutations, PermuteIO{ib1, cs1}, PermuteIO{ib2, cs2})

	var pssi permuteSubStreamInterface
	pssi = &p

	pss := pssi.getSubStream()

	if !reflect.DeepEqual(pss.permutations, round.Permutations) {
		t.Errorf("PermuteStream.Link: Permutation not linked properly")
	}

	compareIntBuffers(grp, ib1, pss.inputs[0], "PermuteStream.Link", t)
	compareIntBuffers(grp, ib2, pss.inputs[1], "PermuteStream.Link", t)

	compareIntSlices(grp, cs1, pss.outputs[0], "PermuteStream.Link", t)
	compareIntSlices(grp, cs2, pss.outputs[1], "PermuteStream.Link", t)
}

func TestPermuteDummyCryptopPrototype_GetInputSize(t *testing.T) {
	if permuteDummyCryptop.GetInputSize() != 1 {
		t.Errorf("PermuteDummyCryptop returned the wrong InputSize, Expected: 1, Recieved: %v", permuteDummyCryptop.GetInputSize())
	}
}

func TestPermuteDummyCryptopPrototype_GetName(t *testing.T) {
	if permuteDummyCryptop.GetInputSize() != 1 {
		t.Errorf("PermuteDummyCryptop returned the wrong Name, Expected: 'Permute Dummy Cryptop', Recieved: %s",
			permuteDummyCryptop.GetName())
	}
}

func compareIntBuffers(grp *cyclic.Group, iba, ibb *cyclic.IntBuffer, source string, t *testing.T) {
	if iba.Len() != ibb.Len() {
		t.Errorf("%s: Int buffers not of same length: A: %v, B: %v ", source, iba.Len(), ibb.Len())
	}

	for i := uint32(0); i < uint32(iba.Len()); i++ {
		if iba.Get(i).Cmp(ibb.Get(i)) != 0 {
			t.Errorf("%s: Int buffers not of same at index %v", source, i)
		}
	}

	for i := uint32(0); i < uint32(iba.Len()); i++ {
		grp.Inverse(iba.Get(i), iba.Get(i))
		if iba.Get(i).Cmp(ibb.Get(i)) != 0 {
			t.Errorf("%s: Int buffers not of same at index %v after modification", source, i)
		}
	}
}

func compareIntSlices(grp *cyclic.Group, iSlcA, iSlcB []*cyclic.Int, source string, t *testing.T) {
	if len(iSlcA) != len(iSlcB) {
		t.Errorf("%s: Int Slices not of same length: A: %v, B: %v ", source, len(iSlcA), len(iSlcB))
	}

	for i := uint32(0); i < uint32(len(iSlcA)); i++ {
		if iSlcA[i].Cmp(iSlcB[i]) != 0 {
			t.Errorf("%s: Int Slices not of same at index %v", source, i)
		}
	}

	for i := uint32(0); i < uint32(len(iSlcA)); i++ {
		grp.Inverse(iSlcA[i], iSlcA[i])
		if iSlcA[i].Cmp(iSlcB[i]) != 0 {
			t.Errorf("%s: Int Slices not of same at index %v after modification", source, i)
		}
	}
}

type PermuteTestStream struct {
	Grp  *cyclic.Group
	in1  *cyclic.IntBuffer
	in2  *cyclic.IntBuffer
	out1 []*cyclic.Int
	out2 []*cyclic.Int

	PermuteSubStream
}

func (s *PermuteTestStream) GetName() string {
	return "PermuteTestStream"
}

func (s *PermuteTestStream) Link(batchSize uint32, source interface{}) {
	round := source.(*node.RoundBuffer)

	s.Grp = round.Grp

	s.in1 = s.Grp.NewIntBuffer(batchSize, s.Grp.NewInt(1))
	s.in2 = s.Grp.NewIntBuffer(batchSize, s.Grp.NewInt(1))

	s.out1 = make([]*cyclic.Int, batchSize)
	s.out2 = make([]*cyclic.Int, batchSize)

	s.PermuteSubStream.LinkStreams(batchSize, round.Permutations,
		PermuteIO{s.in1, s.out1},
		PermuteIO{s.in2, s.out2})
}

func (s *PermuteTestStream) Output(index uint32) *mixmessages.CmixSlot {
	return nil
}
func (s *PermuteTestStream) Input(index uint32, msg *mixmessages.CmixSlot) error {
	return nil
}

func TestPermuteInGraph(t *testing.T) {
	primeString := "FFFFFFFFFFFFFFFFC90FDAA22168C234C4C6628B80DC1CD1" +
		"29024E088A67CC74020BBEA63B139B22514A08798E3404DD" +
		"EF9519B3CD3A431B302B0A6DF25F14374FE1356D6D51C245" +
		"E485B576625E7EC6F44C42E9A637ED6B0BFF5CB6F406B7ED" +
		"EE386BFB5A899FA5AE9F24117C4B1FE649286651ECE45B3D" +
		"C2007CB8A163BF0598DA48361C55D39A69163FA8FD24CF5F" +
		"83655D23DCA3AD961C62F356208552BB9ED529077096966D" +
		"670C354E4ABC9804F1746C08CA18217C32905E462E36CE3B" +
		"E39E772C180E86039B2783A2EC07A28FB5C55DF06F4C52C9" +
		"DE2BCBF6955817183995497CEA956AE515D2261898FA0510" +
		"15728E5A8AACAA68FFFFFFFFFFFFFFFF"
	grp := cyclic.NewGroup(large.NewIntFromString(primeString, 16), large.NewInt(2), large.NewInt(1283))

	batchSize := uint32(10)

	permute := Permute.DeepCopy()

	permuteStream := PermuteTestStream{}

	PanicHandler := func(err error) {
		panic(fmt.Sprintf("Permute: Error in adapter: %s", err.Error()))
		return
	}

	gc := services.NewGraphGenerator(4, PanicHandler, uint8(runtime.NumCPU()))

	g := gc.NewGraph("test", &permuteStream)

	g.First(permute)
	g.Last(permute)

	g.Build(batchSize, services.AUTO_OUTPUTSIZE, 1.0)

	var done *uint32
	done = new(uint32)
	*done = 0

	round := node.NewRound(grp, batchSize, g.GetExpandedBatchSize())

	shuffle.Shuffle32(&round.Permutations)

	g.Link(round)

	permuteInverse := make([]uint32, g.GetExpandedBatchSize())
	for i := uint32(0); i < uint32(len(permuteInverse)); i++ {
		permuteInverse[round.Permutations[i]] = i
	}

	for i := uint32(0); i < batchSize; i++ {
		grp.SetUint64(permuteStream.in1.Get(i), uint64(i+1))
		grp.SetUint64(permuteStream.in2.Get(i), uint64(i+1001))
	}

	g.Run()

	go func(g *services.Graph) {

		for i := uint32(0); i < g.GetBatchSize()-1; i++ {
			g.Send(services.NewChunk(i, i+1))
		}

		atomic.AddUint32(done, 1)
		g.Send(services.NewChunk(g.GetExpandedBatchSize()-1, g.GetExpandedBatchSize()))
	}(g)

	ok := true
	var chunk services.Chunk

	for ok {
		chunk, ok = g.GetOutput()
		for i := chunk.Begin(); i < chunk.End(); i++ {

			d := atomic.LoadUint32(done)

			if d == 0 {
				t.Error("Permute: should not be outputting until all inputs are inputted")
			}
			// Compute expected result for this slot
			if permuteStream.out1[i].Cmp(permuteStream.in1.Get(permuteInverse[i])) != 0 {
				t.Error(fmt.Sprintf("Permute: Slot %v out1 not permuted correctly", i))
			}

			if permuteStream.out2[i].Cmp(permuteStream.in2.Get(permuteInverse[i])) != 0 {
				t.Error(fmt.Sprintf("Permute: Slot %v out2 not permuted correctly", i))
			}
		}
	}
}
