package receivers

import (
	"encoding/json"
	"gitlab.com/elixxir/comms/connect"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/globals"
	"gitlab.com/elixxir/server/server"
	"gitlab.com/elixxir/server/server/measure"
	"gitlab.com/elixxir/server/server/phase"
	"gitlab.com/elixxir/server/server/round"
	"gitlab.com/elixxir/server/server/state"
	"gitlab.com/elixxir/server/testUtil"
	"sync"
	"testing"
	"time"
)

func TestReceiveGetMeasure(t *testing.T) {
	// Smoke tests the management part of PostPrecompResult
	grp := initImplGroup()
	const numNodes = 5
	var err error

	resourceMonitor := measure.ResourceMonitor{}
	resourceMonitor.Set(&measure.ResourceMetric{})
	topology := connect.NewCircuit(buildMockNodeIDs(numNodes))
	// Set instance for first node

	m := state.NewMachine(dummyStates)

	metric := measure.ResourceMetric{
		SystemStartTime: time.Time{},
		Time:            time.Time{},
		MemAllocBytes:   0,
		MemAvailable:    0,
		NumThreads:      0,
		CPUPercentage:   0,
	}

	monitor := measure.ResourceMonitor{LastMetric: &metric, RWMutex: sync.RWMutex{},}
	//nid := server.GenerateId(t)
	def := server.Definition{
		ID:              topology.GetNodeAtIndex(0),
		ResourceMonitor: &monitor,
		UserRegistry:    &globals.UserMap{},
		FullNDF:         testUtil.NDF,
		PartialNDF:      testUtil.NDF,
	}

	instance, _ := server.CreateServerInstance(&def, NewImplementation, m, false)

	// Set up a round first node
	roundID := id.Round(45)

	response := phase.NewResponse(phase.ResponseDefinition{
		PhaseAtSource:  phase.RealPermute,
		ExpectedStates: []phase.State{phase.Active},
		PhaseToExecute: phase.RealPermute})

	responseMap := make(phase.ResponseMap)
	responseMap["RealPermuteVerification"] = response

	p := testUtil.InitMockPhase(t)
	p.Ptype = phase.RealPermute

	rnd := round.New(grp, nil, roundID, []phase.Phase{p}, responseMap, topology,
		topology.GetNodeAtIndex(0), 3, instance.GetRngStreamGen(),
		"0.0.0.0")

	instance.GetRoundManager().AddRound(rnd)

	var resp *mixmessages.RoundMetrics

	info := mixmessages.RoundInfo{
		ID: uint64(roundID),
	}

	rnd.GetMeasurementsReadyChan() <- struct{}{}

	resp, err = ReceiveGetMeasure(instance, &info)

	if err != nil {
		t.Errorf("Failed to return metrics: %+v", err)
	}
	remade := *new(measure.RoundMetrics)

	err = json.Unmarshal([]byte(resp.RoundMetricJSON), &remade)

	if err != nil {
		t.Errorf("Failed to extract data from JSON: %+v", err)
	}

	info = mixmessages.RoundInfo{
		ID: uint64(roundID) - 1,
	}

	_, err = ReceiveGetMeasure(instance, &info)

	if err == nil {
		t.Errorf("This should have thrown an error, instead got: %+v", err)
	}
}
