////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package realtime

import (
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/large"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/globals"
	"gitlab.com/elixxir/server/graphs"
	"gitlab.com/elixxir/server/node"
	"gitlab.com/elixxir/server/server"
	"gitlab.com/elixxir/server/server/round"
	"gitlab.com/elixxir/server/services"
	"os"
	"reflect"
	"runtime"
	"testing"
)

const MODP768 = "FFFFFFFFFFFFFFFFC90FDAA22168C234C4C6628B80DC1CD1" +
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

var prime *large.Int
var grp *cyclic.Group
var batchSize uint32

func TestMain(m *testing.M) {
	prime = large.NewIntFromString(MODP768, 16)
	grp = cyclic.NewGroup(prime, large.NewInt(5), large.NewInt(53))
	batchSize = uint32(100)
	os.Exit(m.Run())
}

// Test that DecryptPermuteStream.GetName() returns the correct name.
func TestDecryptPermuteStream_GetName(t *testing.T) {
	expected := "RealtimeDecryptPermuteStream"

	ps := DecryptPermuteStream{}

	if ps.GetName() != expected {
		t.Errorf("DecryptPermuteStream.GetName(), Expected %s, Recieved %s", expected, ps.GetName())
	}
}

// Test that DecryptPermuteStream.Link() Links correctly.
func TestDecryptPermuteStream_Link(t *testing.T) {
	dps := DecryptPermuteStream{}

	instance := server.CreateServerInstance(grp, &globals.UserMap{})

	roundBuf := round.NewBuffer(grp, 1, batchSize)

	dps.Link(grp, batchSize, roundBuf, instance)

	// DecryptStream Link
	checkIntBuffer(dps.DecryptStream.EcrMsg, batchSize, "EcrMsg",
		grp.NewInt(1), t)
	checkIntBuffer(dps.DecryptStream.EcrAD, batchSize, "EcrAD",
		grp.NewInt(1), t)
	checkIntBuffer(dps.DecryptStream.KeysMsg, batchSize, "KeysMsg",
		grp.NewInt(1), t)
	checkIntBuffer(dps.DecryptStream.KeysAD, batchSize, "KeysAD",
		grp.NewInt(1), t)

	checkStreamIntBuffer(dps.DecryptStream.Grp, dps.DecryptStream.R, roundBuf.R,
		"round.R", t)
	checkStreamIntBuffer(dps.DecryptStream.Grp, dps.DecryptStream.U, roundBuf.U,
		"round.U", t)

	if uint32(len(dps.DecryptStream.Users)) != batchSize {
		t.Errorf("dispatchStream.link(): user slice not created at "+
			"correct length. Expected: %v, Recieved: %v",
			batchSize, len(dps.DecryptStream.Users))
	}

	if uint32(len(dps.DecryptStream.Salts)) != batchSize {
		t.Errorf("dispatchStream.link(): salts slice not created at"+
			" correct length. Expected: %v, Recieved: %v",
			batchSize, len(dps.DecryptStream.Salts))
	}

	for itr, u := range dps.DecryptStream.Users {
		if !reflect.DeepEqual(u, &id.User{}) {
			t.Errorf("dispatchStream.link(): user is at slot %v "+
				"not initilized properly", itr)
		}
	}

	// Permute Link
	checkStreamIntBuffer(grp, dps.PermuteStream.S, roundBuf.S, "S", t)
	checkStreamIntBuffer(grp, dps.PermuteStream.V, roundBuf.V, "V", t)

	checkIntBuffer(dps.PermuteStream.EcrMsg, batchSize, "Msg", grp.NewInt(1), t)
	checkIntBuffer(dps.PermuteStream.EcrAD, batchSize, "AD", grp.NewInt(1), t)

}

// Tests Input's happy path.
func TestDecryptPermuteStream_Input(t *testing.T) {
	dps := &DecryptPermuteStream{}

	instance := server.CreateServerInstance(grp, &globals.UserMap{})

	roundBuf := round.NewBuffer(grp, 1, batchSize)

	dps.Link(grp, batchSize, roundBuf, instance)

	// Only bother checking the input decrypt stream here.
	for b := uint32(0); b < batchSize; b++ {

		expected := [][]byte{
			{byte(b + 1), 0},
			{byte(b + 1), 1},
			make([]byte, 32),
			make([]byte, 32),
		}

		expected[2][0] = byte(b + 1)
		expected[2][1] = byte(2)
		expected[3][0] = byte(b + 1)
		expected[3][1] = byte(3)

		msg := &mixmessages.Slot{
			MessagePayload: expected[0],
			AssociatedData: expected[1],
			SenderID:       expected[2],
			Salt:           expected[3],
		}

		err := dps.DecryptStream.Input(b, msg)
		if err != nil {
			t.Errorf("DecryptStream.Input() errored on slot"+
				" %v: %s", b, err.Error())
		}

		if !reflect.DeepEqual(dps.DecryptStream.EcrMsg.Get(b).Bytes(),
			expected[0]) {
			t.Errorf("DecryptStream.Input() incorrect "+
				"stored EcrMsg data at %v: "+
				"Expected: %v, Recieved: %v",
				b, expected[0],
				dps.DecryptStream.EcrMsg.Get(b).Bytes())
		}

		if !reflect.DeepEqual(dps.DecryptStream.EcrAD.Get(b).Bytes(),
			expected[1]) {
			t.Errorf("DecryptStream.Input() incorrect "+
				" stored EcrAD data at %v: "+
				"Expected: %v, Recieved: %v",
				b, expected[1],
				dps.DecryptStream.EcrAD.Get(b).Bytes())
		}
	}
}

// Tests that the input errors correctly when the index is outside of the batch.
func TestDecryptPermuteStream_Input_OutOfBatch(t *testing.T) {
	dps := &DecryptPermuteStream{}

	instance := server.CreateServerInstance(grp, &globals.UserMap{})

	roundBuf := round.NewBuffer(grp, 1, batchSize)

	dps.Link(grp, batchSize, roundBuf, instance)

	msg := &mixmessages.Slot{
		MessagePayload: []byte{0},
		AssociatedData: []byte{0},
	}

	err := dps.Input(batchSize, msg)

	if err == nil {
		t.Errorf("DecryptStream.Input() did nto return an error " +
			"when out of batch")
	}

	err1 := dps.DecryptStream.Input(batchSize+1, msg)

	if err1 == nil {
		t.Errorf("DecryptStream.Input() did nto return an error " +
			"when out of batch")
	}

}

//Tests that Input errors correct when the passed value is out of the group
func TestDecryptPermuteStream_Input_OutOfGroup(t *testing.T) {
	dps := &DecryptPermuteStream{}

	instance := server.CreateServerInstance(grp, &globals.UserMap{})

	roundBuf := round.NewBuffer(grp, 1, batchSize)

	dps.Link(grp, batchSize, roundBuf, instance)

	msg := &mixmessages.Slot{
		MessagePayload: []byte{0},
		AssociatedData: []byte{0},
	}

	err := dps.Input(batchSize-10, msg)

	if err != node.ErrOutsideOfGroup {
		t.Errorf("DecryptPermuteStream.Input() did not return an " +
			"error when out of group")
	}
}

// Tests that the output function returns a valid cmixMessage.
func TestDecryptPermuteStream_Output(t *testing.T) {
	dps := &DecryptPermuteStream{}

	instance := server.CreateServerInstance(grp, &globals.UserMap{})

	roundBuf := round.NewBuffer(grp, 1, batchSize)

	dps.Link(grp, batchSize, roundBuf, instance)

	// Check only the output side (PermuteStream)
	for b := uint32(0); b < batchSize; b++ {

		expected := [][]byte{
			{byte(b + 1), 0},
			{byte(b + 1), 1},
		}

		dps.PermuteStream.MsgPermuted[b] = grp.NewIntFromBytes(
			expected[0])
		dps.PermuteStream.ADPermuted[b] = grp.NewIntFromBytes(
			expected[1])

		output := dps.Output(b)

		if !reflect.DeepEqual(output.MessagePayload, expected[0]) {
			t.Errorf("PermuteStream.Output() incorrect "+
				"recieved MessagePayload data at %v:"+
				" Expected: %v, Recieved: %v",
				b, expected[0], output.MessagePayload)
		}

		if !reflect.DeepEqual(output.AssociatedData, expected[1]) {
			t.Errorf("PermuteStream.Output() incorrect"+
				" recieved AssociatedData data at %v:"+
				" Expected: %v, Recieved: %v",
				b, expected[1], output.AssociatedData)
		}
	}
}

func TestDecryptPermuteStream_InGraph(t *testing.T) {

	// Create a user registry and make a user in it
	// Unfortunately, this has to time out the db connection before the rest
	// of the test can run. It would be nice to have a method that only makes
	// a user map to make tests run faster
	instance := server.CreateServerInstance(grp, &globals.UserMap{})

	u := instance.GetUserRegistry().NewUser(grp)
	instance.GetUserRegistry().UpsertUser(u)

	// Reception base key should be around 256 bits long,
	// depending on generation, to feed the 256-bit hash
	if u.BaseKey.BitLen() < 250 || u.BaseKey.BitLen() > 256 {
		t.Errorf("Base key has wrong number of bits. "+
			"Had %v bits in reception base key",
			u.BaseKey.BitLen())
	}

	//var stream DecryptStream
	batchSize := uint32(32)
	//stream.Link(batchSize, &node.RoundBuffer{Grp: grp})

	// make a salt for testing
	testSalt := []byte("sodium chloride")
	// pad to length of the base key
	testSalt = append(testSalt, make([]byte, 256/8-len(testSalt))...)

	PanicHandler := func(err error) {
		t.Errorf("Error in adaptor: %s", err.Error())
		return
	}
	// Show that the Init function meets the function type
	var graphInit graphs.Initializer
	graphInit = InitDecryptPermuteGraph

	gc := services.NewGraphGenerator(4, PanicHandler, uint8(runtime.NumCPU()), services.AUTO_OUTPUTSIZE, 1.0)

	//Initialize graph
	g := graphInit(gc)

	g.Build(batchSize)

	// Build the round
	roundBuf := round.NewBuffer(grp, g.GetBatchSize(), g.GetExpandedBatchSize())

	// Fill the fields of the round object for testing
	for i := uint32(0); i < g.GetExpandedBatchSize(); i++ {
		grp.Set(roundBuf.R.Get(i), grp.NewInt(int64(2*i+1)))
		grp.Set(roundBuf.S.Get(i), grp.NewInt(int64(3*i+1)))
		grp.Set(roundBuf.ADPrecomputation.Get(i), grp.NewInt(int64(1)))
		grp.Set(roundBuf.MessagePrecomputation.Get(i), grp.NewInt(int64(1)))

	}

	g.Link(grp, roundBuf, instance)

	// NOTE: Since the math is independently tested, no need to repeat
	// ourselves here.
	stream := g.GetStream().(*DecryptPermuteStream)

	for i := uint32(0); i < g.GetExpandedBatchSize(); i++ {
		copy(stream.Users[i][:], u.ID[:])
		stream.Salts[i] = testSalt
	}

	if stream == nil {
		t.Errorf("Got nil stream instead of a DecryptPermuteStream!")
	}

	g.Run()
	go g.Send(services.NewChunk(0, g.GetExpandedBatchSize()))

	ok := true
	var chunk services.Chunk

	count := uint32(0)
	for ok {
		chunk, ok = g.GetOutput()
		for i := chunk.Begin(); i < chunk.End(); i++ {
			count++
		}
	}

	if count != batchSize {
		t.Errorf("Was not able to send batchSize through! %d < %d",
			count, batchSize)
	}
}
