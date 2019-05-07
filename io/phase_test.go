////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package io

// FIXME: this import list makes it feel like the api is spaghetti
import (
	"fmt"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/comms/connect"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/cryptops"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/large"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/globals"
	"gitlab.com/elixxir/server/server"
	"gitlab.com/elixxir/server/server/phase"
	"gitlab.com/elixxir/server/server/round"
	"gitlab.com/elixxir/server/services"
	"os"
	"testing"
	"time"
)

var nodeIDs *services.NodeIDList

const primeString = "FFFFFFFFFFFFFFFFC90FDAA22168C234C4C6628B80DC1CD1" +
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

var instances []*server.Instance
var grp *cyclic.Group

func TestMain(m *testing.M) {
	// We need 3 servers, prev, cur, next
	addrFmt := "localhost:500%d"
	cnt := 3
	ids := make([]*id.Node, cnt)
	// FIXME: we shouldn't need to do this for a comms test
	grp = cyclic.NewGroup(large.NewIntFromString(primeString, 16),
		large.NewInt(2), large.NewInt(1283))
	instances = make([]*server.Instance, cnt)
	for i := 0; i < cnt; i++ {
		ids[i] = &id.Node{byte(i)}
		// This also seems like overkill for a comms test
		addr := fmt.Sprintf(addrFmt, i)
		instances[i] = server.CreateServerInstance(grp, &globals.UserMap{})
		network := instances[i].InitNetwork(addr, NewServerImplementation, "",
			"")
		defer network.Shutdown()
	}
	// Connect all of the servers to all the other servers
	for connectFrom := 0; connectFrom < cnt; connectFrom++ {
		for connectTo := 0; connectTo < cnt; connectTo++ {
			// don't connect nodes to themselves; communication within a node
			// should, ideally, happen locally
			if connectFrom != connectTo {
				instances[connectFrom].GetNetwork().ConnectToNode(
					ids[connectTo],
					&connect.ConnectionInfo{
						Address: fmt.Sprintf(addrFmt, connectTo),
					})
			}
		}
	}
	nodeIDs = services.NewNodeIDList(ids, 1)
	os.Exit(m.Run())
}

// FIXME: The typedef's on these are.. annoying to have to go lookup

// FIXME: Shouldn't this func type take a parameter?
// FIXME: The semantics of tihs don't make sense and tehre's no comment on what
// it should do.
var chunkCnt = 0

func getChunk() (services.Chunk, bool) {
	if chunkCnt == 0 {
		chunkCnt++
		return services.NewChunk(0, 1), false
	}
	return services.NewChunk(0, 0), true
}

func getMsg(index uint32) *mixmessages.Slot {
	jww.ERROR.Printf("Index %d", index)
	return &mixmessages.Slot{}
}

// Also annoying to implement this everywhere to run tests, but at least it's
// mockable. We need to consider making round more of a data struct and
// implementing it with nil pointers on the stuff we aren't testing/don't need

// Stolen from round_test.go
// FIXME: Dummy impl should probably be made to make tests easier
type mockCryptop struct{}

func (*mockCryptop) GetName() string      { return "mockCryptop" }
func (*mockCryptop) GetInputSize() uint32 { return 1 }

type mockStream struct{}

func (*mockStream) Input(uint32, *mixmessages.Slot) error { return nil }
func (*mockStream) Output(uint32) *mixmessages.Slot       { return nil }
func (*mockStream) GetName() string {
	return "mockStream"
}
func (*mockStream) Link(*cyclic.Group, uint32, ...interface{}) {}

func initMockGraph(gg services.GraphGenerator) *services.Graph {
	graph := gg.NewGraph("MockGraph", &mockStream{})
	var mockModule services.Module
	mockModule.Adapt = func(stream services.Stream,
		cryptop cryptops.Cryptop, chunk services.Chunk) error {
		return nil
	}
	mockModule.Cryptop = &mockCryptop{}
	mockModuleCopy := mockModule.DeepCopy()
	graph.First(mockModuleCopy)
	graph.Last(mockModuleCopy)
	graph.Build(1)
	return graph
}

func TestPostPhase(t *testing.T) {
	testPhase := phase.New(initMockGraph(services.
		NewGraphGenerator(1, nil, 1, 1, 1)),
		phase.RealPermute, TransmitPhase, time.Second)
	// We didn't set everything up, so we stop at PostPhase error on the
	// RoundManager GetPhase function
	// Comment out for now.
	/*
		err := TransmitPhase(1, 42, phase.RealPermute, getChunk, getMsg,
			nodeIDs)

		if err != nil {
			t.Errorf("%v", err)
		}
	*/
	// Now fix the round manager
	rm := instances[2].GetRoundManager()
	roundID := id.Round(42)

	phases := make([]*phase.Phase, 1)
	phases[0] = testPhase

	thisRound := round.New(grp, roundID, phases, nil, 0, 1)
	rm.AddRound(thisRound)
	// Reset get chunk
	chunkCnt = 0
	// Is this the right instance out of the three to use?
	err := TransmitPhase(instances[0].GetNetwork(), 1, 42, phase.RealPermute, getChunk,
		getMsg, nodeIDs)

	if err != nil {
		t.Errorf("%v", err)
	}
}
