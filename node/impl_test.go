package node

import (
	"fmt"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/cryptops"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/large"
	"gitlab.com/elixxir/primitives/circuit"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/globals"
	"gitlab.com/elixxir/server/server"
	"gitlab.com/elixxir/server/server/phase"
	"gitlab.com/elixxir/server/server/round"
	"gitlab.com/elixxir/server/services"
	"runtime"
	"testing"
	"time"
)

func TestNewImplementation_PostPhase(t *testing.T) {
	batchSize := uint32(11)
	roundID := id.Round(0)

	grp := initImplGroup()
	instance := server.CreateServerInstance(grp, &globals.UserMap{})
	mockPhase := initMockPhase()

	responceMap := make(phase.ResponseMap)
	responceMap[mockPhase.GetType().String()] =
		phase.NewResponse(mockPhase.GetType(), mockPhase.GetType(),
			phase.Available)

	topology := buildMockTopology(2)

	r := round.New(grp, roundID, []phase.Phase{mockPhase}, responceMap,
		topology, topology.GetNodeAtIndex(0), batchSize)

	instance.GetRoundManager().AddRound(r)

	fmt.Println()

	//get the impl
	impl := NewImplementation(instance, time.Second)

	//Build a mock mockBatch to receive
	mockBatch := &mixmessages.Batch{}

	for i := uint32(0); i < batchSize; i++ {
		mockBatch.Slots = append(mockBatch.Slots,
			&mixmessages.Slot{
				MessagePayload: []byte{byte(i)},
			})
	}

	mockBatch.ForPhase = int32(mockPhase.GetType())
	mockBatch.Round = &mixmessages.RoundInfo{ID: uint64(roundID)}

	//send the mockBatch to the impl
	impl.PostPhase(mockBatch)

	//check the mock phase to see if the correct result has been stored
	for index := range mockBatch.Slots {
		if mockPhase.chunks[index].Begin() != uint32(index) {
			t.Errorf("PostPhase: output chunk not equal to passed;"+
				"Expected: %v, Recieved: %v", index, mockPhase.chunks[index].Begin())
		}

		if mockPhase.indices[index] != uint32(index) {
			t.Errorf("PostPhase: output index  not equal to passed;"+
				"Expected: %v, Recieved: %v", index, mockPhase.indices[index])
		}
	}
}

/*Mock Graph*/
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

/*Mock Phase*/
type MockPhase struct {
	graph        *services.Graph
	chunks       []services.Chunk
	indices      []uint32
	stateChecker phase.GetState
}

func (mp *MockPhase) Send(chunk services.Chunk) {
	mp.chunks = append(mp.chunks, chunk)
}

func (mp *MockPhase) Input(index uint32, slot *mixmessages.Slot) error {
	if len(slot.Salt) != 0 {
		return errors.New("error to test edge case")
	}
	mp.indices = append(mp.indices, index)
	return nil
}

func (mp *MockPhase) ConnectToRound(id id.Round, setState phase.Transition,
	getState phase.GetState) {
	mp.stateChecker = getState
	return
}

func (mp *MockPhase) GetState() phase.State     { return mp.stateChecker() }
func (mp *MockPhase) GetGraph() *services.Graph { return mp.graph }

func (*MockPhase) EnableVerification()                    { return }
func (*MockPhase) GetRoundID() id.Round                   { return 0 }
func (*MockPhase) GetType() phase.Type                    { return 0 }
func (*MockPhase) AttemptTransitionToQueued() bool        { return false }
func (*MockPhase) TransitionToRunning()                   { return }
func (*MockPhase) UpdateFinalStates() bool                { return false }
func (*MockPhase) GetTransmissionHandler() phase.Transmit { return nil }
func (*MockPhase) GetTimeout() time.Duration              { return 0 }
func (*MockPhase) Cmp(phase.Phase) bool                   { return false }
func (*MockPhase) String() string                         { return "" }

func initMockPhase() *MockPhase {
	gc := services.NewGraphGenerator(1, nil, uint8(runtime.NumCPU()), services.AutoOutputSize, 0)
	g := gc.NewGraph("MockGraph", &mockStream{})
	var mockModule services.Module
	mockModule.Adapt = func(stream services.Stream,
		cryptop cryptops.Cryptop, chunk services.Chunk) error {
		return nil
	}
	mockModule.Cryptop = &mockCryptop{}
	mockModuleCopy := mockModule.DeepCopy()
	g.First(mockModuleCopy)
	g.Last(mockModuleCopy)
	return &MockPhase{graph: g}
}

func initImplGroup() *cyclic.Group {
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
	return grp
}

func buildMockTopology(numNodes int) *circuit.Circuit {
	var nodeIDs []*id.Node

	//Build IDs
	for i := 0; i < numNodes; i++ {
		nodeID := &id.Node{}
		nodIDBytes := make([]byte, id.NodeIdLen)
		nodIDBytes[0] = byte(i + 1)
		nodeID.SetBytes(nodIDBytes)
		nodeIDs = append(nodeIDs, nodeID)
	}

	//Build the topology
	return circuit.New(nodeIDs)
}
