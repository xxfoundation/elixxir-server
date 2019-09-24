package testUtil

import (
	"errors"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/cryptops"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/server/measure"
	"gitlab.com/elixxir/server/server/phase"
	"gitlab.com/elixxir/server/services"
	"runtime"
	"time"
)

/* Mock Graph */
type mockCryptop struct{}

func (*mockCryptop) GetName() string      { return "mockCryptop" }
func (*mockCryptop) GetInputSize() uint32 { return 1 }

type MockStream struct{}

func (*MockStream) Input(uint32, *mixmessages.Slot) error { return nil }
func (*MockStream) Output(uint32) *mixmessages.Slot       { return nil }
func (*MockStream) GetName() string {
	return "MockStream"
}
func (*MockStream) Link(*cyclic.Group, uint32, ...interface{}) {}

/*Mock Phase*/
type MockPhase struct {
	graph        *services.Graph
	chunks       []services.Chunk
	indices      []uint32
	stateChecker phase.GetState
	Ptype        phase.Type
}

func (mp *MockPhase) GetChunks() []services.Chunk {
	return mp.chunks
}

func (mp *MockPhase) GetIndices() []uint32 {
	return mp.indices
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

func (*MockPhase) EnableVerification()    { return }
func (*MockPhase) GetRoundID() id.Round   { return 0 }
func (mp *MockPhase) GetType() phase.Type { return mp.Ptype }
func (mp *MockPhase) AttemptToQueue(queue chan<- phase.Phase) bool {
	queue <- mp
	return true
}
func (mp *MockPhase) IsQueued() bool                      { return true }
func (*MockPhase) UpdateFinalStates()                     { return }
func (*MockPhase) GetTransmissionHandler() phase.Transmit { return nil }
func (*MockPhase) GetTimeout() time.Duration              { return 0 }
func (*MockPhase) Cmp(phase.Phase) bool                   { return false }
func (*MockPhase) String() string                         { return "" }
func (*MockPhase) Measure(string)                         { return }
func (*MockPhase) GetMeasure() measure.Metrics            { return *new(measure.Metrics) }

func InitMockPhase() *MockPhase {
	gc := services.NewGraphGenerator(1, nil, uint8(runtime.NumCPU()), services.AutoOutputSize, 0)
	g := gc.NewGraph("MockGraph", &MockStream{})
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
