package receivers

import (
	"fmt"
	"gitlab.com/elixxir/comms/connect"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/globals"
	"gitlab.com/elixxir/server/graphs/realtime"
	"gitlab.com/elixxir/server/server"
	"gitlab.com/elixxir/server/server/measure"
	"gitlab.com/elixxir/server/server/phase"
	"gitlab.com/elixxir/server/server/round"
	"gitlab.com/elixxir/server/server/state"
	"gitlab.com/elixxir/server/services"
	"gitlab.com/elixxir/server/testUtil"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestReceivePostNewBatch_Errors(t *testing.T) {
	// This round should be at a state where its precomp is complete.
	// So, we might want more than one phase,
	// since it's at a boundary between phases.
	grp := initImplGroup()

	topology := connect.NewCircuit(buildMockNodeIDs(5))
	def := server.Definition{
		UserRegistry:    &globals.UserMap{},
		ResourceMonitor: &measure.ResourceMonitor{},
	}
	def.ID = topology.GetNodeAtIndex(0)
	def.Gateway.ID = id.NewTmpGateway()

	l := [current.NUM_STATES]state.Change{}
	instance, _ := server.CreateServerInstance(&def, NewImplementation, l, false)

	const batchSize = 1
	const roundID = 2

	// Does the mockPhase move through states?
	precompReveal := testUtil.InitMockPhase(t)
	precompReveal.Ptype = phase.PrecompReveal
	realDecrypt := testUtil.InitMockPhase(t)
	realDecrypt.Ptype = phase.RealDecrypt

	tagKey := realDecrypt.Ptype.String()
	responseMap := make(phase.ResponseMap)
	responseMap[tagKey] = phase.NewResponse(phase.ResponseDefinition{
		PhaseAtSource:  realDecrypt.GetType(),
		PhaseToExecute: realDecrypt.GetType(),
		ExpectedStates: []phase.State{phase.Active},
	})

	// Well, this round needs to at least be on the precomp queue?
	// If it's not on the precomp queue,
	// that would let us test the error being returned.
	r := round.New(grp, instance.GetUserRegistry(), roundID,
		[]phase.Phase{precompReveal, realDecrypt}, responseMap, topology,
		topology.GetNodeAtIndex(0), batchSize, instance.GetRngStreamGen(),
		"0.0.0.0")
	instance.GetRoundManager().AddRound(r)

	// Build a fake batch for the reception handler
	// This emulates what the gateway would send to the comm
	batch := &mixmessages.Batch{
		Round: &mixmessages.RoundInfo{
			ID: roundID,
		},
		FromPhase: int32(phase.RealDecrypt),
		Slots: []*mixmessages.Slot{
			{
				// Do the fields need to be populated?
				SenderID: nil,
				PayloadA: nil,
				PayloadB: nil,
				Salt:     nil,
				KMACs:    nil,
			},
		},
	}

	h, _ := connect.NewHost(instance.GetGateway().String(), "test", nil, false, false)
	auth := &connect.Auth{
		IsAuthenticated: true,
		Sender:          h,
	}

	err := ReceivePostNewBatch(instance, batch, auth)
	if err == nil {
		t.Error("ReceivePostNewBatch should have errored out if there were no" +
			" precomputations available")
	}

	// OK, let's put that round on the queue of completed precomps now,
	// which should cause the reception handler to function normally.
	// This should panic because the expected states aren't populated correctly,
	// so the realtime can't continue to be processed.
	defer func() {
		panicResult := recover()
		panicString := panicResult.(string)
		if panicString == "" {
			t.Error("There was no panicked error from the HandleIncomingComm" +
				" call")
		}
	}()
	instance.GetCompletedPrecomps().Push(r)

	h, _ = connect.NewHost(instance.GetGateway().String(), "test", nil, false, false)
	auth = &connect.Auth{
		IsAuthenticated: true,
		Sender:          h,
	}
	err = ReceivePostNewBatch(instance, batch, auth)
}

// Test error case in which sender of postnewbatch is not authenticated
func TestReceivePostNewBatch_AuthError(t *testing.T) {
	instance := mockServerInstance(t)

	const roundID = 2

	batch := &mixmessages.Batch{
		Round: &mixmessages.RoundInfo{
			ID: roundID,
		},
		FromPhase: int32(phase.RealDecrypt),
		Slots: []*mixmessages.Slot{
			{
				// Do the fields need to be populated?
				SenderID: nil,
				PayloadA: nil,
				PayloadB: nil,
				Salt:     nil,
				KMACs:    nil,
			},
		},
	}

	h, _ := connect.NewHost(instance.GetGateway().String(), "test", nil, false, false)
	auth := &connect.Auth{
		IsAuthenticated: false,
		Sender:          h,
	}

	err := ReceivePostNewBatch(instance, batch, auth)

	if err == nil {
		t.Error("Did not receive expected error")
		return
	}

	if !strings.Contains(err.Error(), "Failed to authenticate") {
		t.Error("Did not receive expected authentication error")
	}
}

// Test error case in which the sender of postnewbatch is not who we expect
func TestReceivePostNewBatch_BadSender(t *testing.T) {
	instance := mockServerInstance(t)

	const roundID = 2

	batch := &mixmessages.Batch{
		Round: &mixmessages.RoundInfo{
			ID: roundID,
		},
		FromPhase: int32(phase.RealDecrypt),
		Slots: []*mixmessages.Slot{
			{
				// Do the fields need to be populated?
				SenderID: nil,
				PayloadA: nil,
				PayloadB: nil,
				Salt:     nil,
				KMACs:    nil,
			},
		},
	}

	h, _ := connect.NewHost("test", "test", nil, false, false)
	auth := &connect.Auth{
		IsAuthenticated: true,
		Sender:          h,
	}

	err := ReceivePostNewBatch(instance, batch, auth)

	if err == nil {
		t.Error("Did not receive expected error")
		return
	}

	if !strings.Contains(err.Error(), "Failed to authenticate") {
		t.Error("Did not receive expected authentication error")
	}
}

// Tests the happy path of ReceivePostNewBatch, demonstrating that it can start
// realtime processing with a new batch from the gateway.
// Note: In this case, the happy path includes an error from one of the slots
// that has cryptographically incorrect data.
func TestReceivePostNewBatch(t *testing.T) {
	grp := initImplGroup()

	def := server.Definition{
		CmixGroup:       grp,
		Topology:        connect.NewCircuit(buildMockNodeIDs(1)),
		UserRegistry:    &globals.UserMap{},
		ResourceMonitor: &measure.ResourceMonitor{},
	}

	def.ID = def.Topology.GetNodeAtIndex(0)
	def.Gateway.ID = id.NewTmpGateway()
	instance, _ := server.CreateServerInstance(&def, NewImplementation, false)
	instance.InitFirstNode()
	topology := instance.GetTopology()
	registry := instance.GetUserRegistry()

	// Make and register a user
	sender := registry.NewUser(grp)
	registry.UpsertUser(sender)

	const batchSize = 1
	const roundID = 2

	PanicHandler := func(g, m string, err error) {
		panic(fmt.Sprintf("Error in module %s of graph %s: %s", g, m, err.Error()))
	}
	gg := services.NewGraphGenerator(4, PanicHandler, uint8(runtime.NumCPU()),
		1, 1.0)

	realDecrypt := phase.New(phase.Definition{
		Graph: realtime.InitDecryptGraph(gg),
		Type:  phase.RealDecrypt,
		TransmissionHandler: func(network *node.Comms, batchSize uint32, roundID id.Round, phaseTy phase.Type, getChunk phase.GetChunk, getMessage phase.GetMessage, topology *connect.Circuit, nodeId *id.Node, measure phase.Measure) error {
			return nil
		},
		Timeout:        5 * time.Second,
		DoVerification: false,
	})

	tagKey := realDecrypt.GetType().String()
	responseMap := make(phase.ResponseMap)
	responseMap[tagKey] = phase.NewResponse(phase.ResponseDefinition{
		PhaseAtSource:  realDecrypt.GetType(),
		ExpectedStates: []phase.State{phase.Active},
		PhaseToExecute: realDecrypt.GetType(),
	})

	// We need this round to be on the precomp queue
	r := round.New(grp, instance.GetUserRegistry(), roundID,
		[]phase.Phase{realDecrypt}, responseMap, topology,
		topology.GetNodeAtIndex(0), batchSize, instance.GetRngStreamGen(),
		"0.0.0.0")
	instance.GetRoundManager().AddRound(r)
	instance.GetCompletedPrecomps().Push(r)

	// Build a fake batch for the reception handler
	// This emulates what the gateway would send to the comm
	batch := &mixmessages.Batch{
		Round: &mixmessages.RoundInfo{
			ID: roundID,
		},
		FromPhase: int32(phase.RealDecrypt),
		Slots: []*mixmessages.Slot{
			{
				// Do the fields need to be populated?
				// Yes, but only to check if the batch made it to the phase
				SenderID: sender.ID.Bytes(),
				PayloadA: []byte{2},
				PayloadB: []byte{3},
				// Because the salt is just one byte,
				// this should fail in the Realtime Decrypt graph.
				Salt:  make([]byte, 32),
				KMACs: [][]byte{{5}},
			},
		},
	}

	h, _ := connect.NewHost(instance.GetGateway().String(), "test", nil, false, false)
	auth := &connect.Auth{
		IsAuthenticated: true,
		Sender:          h,
	}

	// Actually, this should return an error because the batch has a malformed
	// slot in it, so once we implement per-slot errors we can test all the
	// realtime decrypt error cases from this reception handler if we want
	err := ReceivePostNewBatch(instance, batch, auth)
	if err != nil {
		t.Error(err)
	}

	// We verify that the Realtime Decrypt phase has been enqueued
	if !realDecrypt.IsQueued() {
		t.Errorf("Realtime decrypt is not queued")
	}
}

