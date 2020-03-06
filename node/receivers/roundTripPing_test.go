package receivers

import (
	"crypto/rand"
	"github.com/golang/protobuf/ptypes"
	"gitlab.com/elixxir/comms/connect"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/globals"
	"gitlab.com/elixxir/server/server"
	"gitlab.com/elixxir/server/server/measure"
	"gitlab.com/elixxir/server/server/phase"
	"gitlab.com/elixxir/server/server/round"
	"gitlab.com/elixxir/server/testUtil"
	"testing"
)

func TestReceiveRoundTripPing(t *testing.T) {
	grp := initImplGroup()
	const numNodes = 5

	resourceMonitor := measure.ResourceMonitor{}
	resourceMonitor.Set(&measure.ResourceMetric{})

	topology := connect.NewCircuit(buildMockNodeIDs(numNodes))
	// Set instance for first node
	def := server.Definition{
		UserRegistry:    &globals.UserMap{},
		ResourceMonitor: &resourceMonitor,
	}
	def.ID = topology.GetNodeAtIndex(0)

	instance, _ := mockServerInstance(t)

	// Set up a round first node
	roundID := id.Round(45)

	mockPhase := testUtil.InitMockPhase(t)
	mockPhase.Ptype = phase.PrecompShare

	tagKey := mockPhase.GetType().String() + "Verification"
	responseMap := make(phase.ResponseMap)
	responseMap[tagKey] = phase.NewResponse(
		phase.ResponseDefinition{
			PhaseAtSource:  mockPhase.GetType(),
			ExpectedStates: []phase.State{phase.Active},
			PhaseToExecute: mockPhase.GetType()},
	)

	batchSize := uint32(11)

	r := round.New(grp, &globals.UserMap{}, roundID, []phase.Phase{mockPhase},
		responseMap, topology, topology.GetNodeAtIndex(0), batchSize,
		instance.GetRngStreamGen(), "0.0.0.0")
	r.StartRoundTrip("test")

	before := r.GetRTEnd().String()

	instance.GetRoundManager().AddRound(r)
	A := make([]byte, 150)
	B := make([]byte, 150)
	_, err := rand.Read(A)
	if err != nil {
		t.Errorf("TransmitRoundTripPing: failed to generate random bytes A: %+v", err)
	}
	_, err = rand.Read(B)
	if err != nil {
		t.Errorf("TransmitRoundTripPing: failed to generate random bytes B: %+v", err)
	}
	anyPayload := &mixmessages.Batch{
		Slots: []*mixmessages.Slot{
			{
				SenderID: instance.GetID().Bytes(),
				PayloadA: A,
				PayloadB: B,
				// Because the salt is just one byte,
				// this should fail in the Realtime Decrypt graph.
				Salt:  make([]byte, 32),
				KMACs: [][]byte{{5}},
			},
		},
	}
	any, _ := ptypes.MarshalAny(anyPayload)

	msg := &mixmessages.RoundTripPing{
		Payload: any,
		Round: &mixmessages.RoundInfo{
			ID: 45,
		},
	}

	err = ReceiveRoundTripPing(instance, msg)
	if err != nil {
		t.Errorf("ReceiveRoundTripPing returned an error: %+v", err)
	}

	if before == r.GetRTEnd().String() {
		t.Error("ReceiveRoundTripPing didn't update end time")
	}
}
