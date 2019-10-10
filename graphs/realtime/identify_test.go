////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package realtime

import (
	"fmt"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/cryptops"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/large"
	"gitlab.com/elixxir/crypto/shuffle"
	"gitlab.com/elixxir/server/server/round"
	"gitlab.com/elixxir/server/services"
	"reflect"
	"runtime"
	"sync/atomic"
	"testing"
)

// Test that IdentifyStream.GetName() returns the correct name.
func TestIdentifyStream_GetName(t *testing.T) {
	expected := "RealtimeIdentifyStream"

	is := IdentifyStream{}

	if is.GetName() != expected {
		t.Errorf("IdentifyStream.GetName(), Expected %s, Recieved %s", expected, is.GetName())
	}
}

// Test that IdentifyStream.Link() Links correctly.
func TestIdentifyStream_Link(t *testing.T) {
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
	grp := cyclic.NewGroup(large.NewIntFromString(primeString, 16),
		large.NewInt(2))

	is := IdentifyStream{}

	batchSize := uint32(100)

	r := round.NewBuffer(grp, batchSize, batchSize)

	is.Link(grp, batchSize, r)

	checkIntBuffer(is.EcrPayloadA, batchSize, "EcrPayloadA", grp.NewInt(1), t)
	checkIntBuffer(is.EcrPayloadB, batchSize, "EcrPayloadB", grp.NewInt(1), t)

	checkStreamIntBuffer(grp, is.PayloadAPrecomputation, r.PayloadAPrecomputation,
		"PayloadAPrecomputation", t)
	checkStreamIntBuffer(grp, is.PayloadBPrecomputation, r.PayloadBPrecomputation,
		"PayloadBPrecomputation", t)
}

// Tests Input's happy path.
func TestIdentifyStream_Input(t *testing.T) {
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
	grp := cyclic.NewGroup(large.NewIntFromString(primeString, 16),
		large.NewInt(2))

	batchSize := uint32(100)

	is := &IdentifyStream{}

	roundBuffer := round.NewBuffer(grp, batchSize, batchSize)

	is.Link(grp, batchSize, roundBuffer)

	for b := uint32(0); b < batchSize; b++ {

		expected := [][]byte{
			{byte(b + 1), 0},
			{byte(b + 1), 1},
		}

		msg := &mixmessages.Slot{
			PayloadA: expected[0],
			PayloadB: expected[1],
		}

		err := is.Input(b, msg)
		if err != nil {
			t.Errorf("IdentifyStream.Input() errored on slot %v: %s", b, err.Error())
		}

		if !reflect.DeepEqual(is.EcrPayloadA.Get(b).Bytes(), expected[0]) {
			t.Errorf("IdentifyStream.Input() incorrect stored EcrPayloadA data at %v: Expected: %v, Recieved: %v",
				b, expected[0], is.EcrPayloadA.Get(b).Bytes())
		}

		if !reflect.DeepEqual(is.EcrPayloadB.Get(b).Bytes(), expected[1]) {
			t.Errorf("IdentifyStream.Input() incorrect stored EcrPayloadB data at %v: Expected: %v, Recieved: %v",
				b, expected[1], is.EcrPayloadB.Get(b).Bytes())
		}
	}
}

// Tests that the input errors correctly when the index is outside of the batch.
func TestIdentifyStream_Input_OutOfBatch(t *testing.T) {
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
	grp := cyclic.NewGroup(large.NewIntFromString(primeString, 16),
		large.NewInt(2))

	batchSize := uint32(100)

	is := &IdentifyStream{}

	roundBuffer := round.NewBuffer(grp, batchSize, batchSize)

	is.Link(grp, batchSize, roundBuffer)

	msg := &mixmessages.Slot{
		PayloadA: []byte{0},
		PayloadB: []byte{0},
	}

	err := is.Input(batchSize, msg)

	if err == nil {
		t.Errorf("IdentifyStream.Input() did nto return an error when out of batch")
	}

	err1 := is.Input(batchSize+1, msg)

	if err1 == nil {
		t.Errorf("IdentifyStream.Input() did nto return an error when out of batch")
	}
}

//Tests that Input errors correct when the passed value is out of the group
func TestIdentifyStream_Input_OutOfGroup(t *testing.T) {
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
	grp := cyclic.NewGroup(large.NewIntFromString(primeString, 16),
		large.NewInt(2))

	batchSize := uint32(100)

	stream := &IdentifyStream{}

	roundBuffer := round.NewBuffer(grp, batchSize, batchSize)

	stream.Link(grp, batchSize, roundBuffer)

	msg := &mixmessages.Slot{
		PayloadA: []byte{0},
		PayloadB: []byte{0},
	}

	err := stream.Input(batchSize-10, msg)

	if err != services.ErrOutsideOfGroup {
		t.Errorf("IdentifyStream.Input() did not return an error when out of group")
	}
}

// Tests that the output function returns a valid cmixMessage.
func TestIdentifyStream_Output(t *testing.T) {
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
	grp := cyclic.NewGroup(large.NewIntFromString(primeString, 16),
		large.NewInt(2))

	batchSize := uint32(100)

	is := &IdentifyStream{}

	roundBuffer := round.NewBuffer(grp, batchSize, batchSize)

	is.Link(grp, batchSize, roundBuffer)

	for b := uint32(0); b < batchSize; b++ {

		expected := [][]byte{
			{byte(b + 1), 0},
			{byte(b + 1), 1},
		}

		is.EcrPayloadAPermuted[b] = grp.NewIntFromBytes(expected[0])
		is.EcrPayloadBPermuted[b] = grp.NewIntFromBytes(expected[1])

		output := is.Output(b)

		if !reflect.DeepEqual(output.PayloadA, grp.NewIntFromBytes(expected[0]).LeftpadBytes(uint64(grp.GetP().ByteLen()))) {
			t.Errorf("IdentifyStream.Output() incorrect recieved PayloadA data at %v: Expected: %v, Recieved: %v",
				b, expected[0], output.PayloadA)
		}

		if !reflect.DeepEqual(output.PayloadB, grp.NewIntFromBytes(expected[1]).LeftpadBytes(uint64(grp.GetP().ByteLen()))) {
			t.Errorf("IdentifyStream.Output() incorrect recieved PayloadB data at %v: Expected: %v, Recieved: %v",
				b, expected[1], output.PayloadB)
		}
	}
}

// Tests that IdentifyStream conforms to the CommsStream interface.
func TestIdentifyStream_CommsInterface(t *testing.T) {
	var face interface{}
	face = &IdentifyStream{}
	_, ok := face.(services.Stream)

	if !ok {
		t.Errorf("IdentifyStream: Does not conform to the CommsStream interface")
	}
}

func TestIdentifyStream_InGraph(t *testing.T) {
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
	grp := cyclic.NewGroup(large.NewIntFromString(primeString, 16),
		large.NewInt(2))

	batchSize := uint32(10)

	PanicHandler := func(g, m string, err error) {
		panic(fmt.Sprintf("Error in module %s of graph %s: %s", g, m, err.Error()))
	}

	gc := services.NewGraphGenerator(4, PanicHandler, uint8(runtime.NumCPU()), 1, 1.0)

	g := InitIdentifyGraph(gc)

	g.Build(batchSize)

	var done *uint32
	done = new(uint32)
	*done = 0

	roundBuffer := round.NewBuffer(grp, batchSize, g.GetExpandedBatchSize())

	subPermutation := roundBuffer.Permutations[:batchSize]

	shuffle.Shuffle32(&subPermutation)

	g.Link(grp, roundBuffer)

	permuteInverse := make([]uint32, g.GetExpandedBatchSize())
	for i := uint32(0); i < uint32(len(permuteInverse)); i++ {
		permuteInverse[roundBuffer.Permutations[i]] = i
	}

	is := g.GetStream().(*IdentifyStream)

	expectedPayloadA := grp.NewIntBuffer(batchSize, grp.NewInt(1))
	expectedPayloadB := grp.NewIntBuffer(batchSize, grp.NewInt(1))

	for i := uint32(0); i < batchSize; i++ {
		grp.SetUint64(is.EcrPayloadA.Get(i), uint64(i+1))
		grp.SetUint64(is.EcrPayloadB.Get(i), uint64(i+1001))
	}

	for i := uint32(0); i < batchSize; i++ {
		grp.Set(expectedPayloadA.Get(i), is.EcrPayloadA.Get(i))
		grp.Set(expectedPayloadB.Get(i), is.EcrPayloadB.Get(i))

		cryptops.Mul2(grp, is.S.Get(i), expectedPayloadA.Get(i))
		cryptops.Mul2(grp, is.V.Get(i), expectedPayloadB.Get(i))
	}

	g.Run()

	go func(g *services.Graph) {

		for i := uint32(0); i < g.GetBatchSize()-1; i++ {
			g.Send(services.NewChunk(i, i+1), nil)
		}

		atomic.AddUint32(done, 1)
		g.Send(services.NewChunk(g.GetExpandedBatchSize()-1, g.GetExpandedBatchSize()), nil)
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
			if is.PayloadAPermuted[i].Cmp(expectedPayloadA.Get(permuteInverse[i])) != 0 {
				t.Error(fmt.Sprintf("Permute: Slot %v out1 not permuted correctly", i))
			}

			if is.PayloadBPermuted[i].Cmp(expectedPayloadB.Get(permuteInverse[i])) != 0 {
				t.Error(fmt.Sprintf("Permute: Slot %v out2 not permuted correctly", i))
			}
		}
	}
}
