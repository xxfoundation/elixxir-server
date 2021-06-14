///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package node

import (
	"fmt"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/server/graphs"
	"gitlab.com/elixxir/server/internal"
	"gitlab.com/elixxir/server/internal/measure"
	"gitlab.com/elixxir/server/internal/phase"
	"gitlab.com/elixxir/server/internal/round"
	"gitlab.com/elixxir/server/internal/state"
	"gitlab.com/elixxir/server/io"
	"gitlab.com/elixxir/server/services"
	"gitlab.com/elixxir/server/testUtil"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/comms/messages"
	"gitlab.com/xx_network/primitives/id"
	"runtime"
	"testing"
	"time"
)

func setup(t *testing.T) (*internal.Instance, *connect.Circuit) {
	var nodeIDs []*id.ID
	//Build IDs
	for i := 0; i < 5; i++ {
		nodIDBytes := make([]byte, id.ArrIDLen)
		nodIDBytes[0] = byte(i + 1)
		nodeID := id.NewIdFromBytes(nodIDBytes, t)
		nodeIDs = append(nodeIDs, nodeID)
	}

	//Build the topology
	topology := connect.NewCircuit(nodeIDs)
	gg := services.NewGraphGenerator(4, 1,
		services.AutoOutputSize, 1.0)
	def := internal.Definition{
		ResourceMonitor:    &measure.ResourceMonitor{},
		FullNDF:            testUtil.NDF,
		PartialNDF:         testUtil.NDF,
		GraphGenerator:     gg,
		RecoveredErrorPath: "/tmp/recovered_error",
		Gateway: internal.GW{
			Address: "0.0.0.0:11420",
		},
		ListeningAddress: "0.0.0.0:11421",
	}
	def.ID = topology.GetNodeAtIndex(0)
	def.Gateway.ID = def.ID.DeepCopy()
	def.Gateway.ID.SetType(id.Gateway)
	def.DevMode = true
	var instance *internal.Instance
	var dummyStates = [current.NUM_STATES]state.Change{
		func(from current.Activity, err *mixmessages.RoundError) error { return nil },
		func(from current.Activity, err *mixmessages.RoundError) error { return nil },
		func(from current.Activity, err *mixmessages.RoundError) error { return nil },
		func(from current.Activity, err *mixmessages.RoundError) error { return nil },
		func(from current.Activity, err *mixmessages.RoundError) error { return nil },
		func(from current.Activity, err *mixmessages.RoundError) error { return nil },
		func(from current.Activity, err *mixmessages.RoundError) error { return nil },
		func(from current.Activity, err *mixmessages.RoundError) error { return nil },
	}
	m := state.NewTestMachine(dummyStates, current.PRECOMPUTING, t)
	fmt.Println("1")
	instance, _ = internal.CreateServerInstance(&def, io.NewImplementation,
		m, "1.1.0")
	fmt.Println("2")
	params := connect.GetDefaultHostParams()
	params.AuthEnabled = false
	_, err := instance.GetNetwork().AddHost(&id.Permissioning, testUtil.NDF.Registration.Address,
		[]byte(testUtil.RegCert), params)
	if err != nil {
		t.Errorf("Failed to add permissioning host: %+v", err)
	}
	fmt.Println("04")
	r := round.NewDummyRoundWithTopology(id.Round(1), 3, topology, t)
	instance.GetRoundManager().AddRound(r)
	_ = instance.Run()
	fmt.Println("05")
	return instance, topology
}

func TestNewStateChanges(t *testing.T) {
	ourStates := NewStateChanges()
	if len(ourStates) != int(current.NUM_STATES) {
		t.Errorf("Length of state table is not of expected length: "+
			"\n\tExpected: %+v"+
			"\n\tReceived: %+v", int(current.NUM_STATES), ourStates)
	}

	for i := 0; i < int(current.NUM_STATES); i++ {
		if ourStates[i] == nil {
			t.Errorf("Case %d wasn't initialized, should not be nil!", i)
		}

	}
}

func TestError(t *testing.T) {
	instance, topology := setup(t)
	rndErr := &mixmessages.RoundError{
		Id:     1,
		NodeId: instance.GetID().Marshal(),
		Error:  "",
	}
	mockBroadcast := func(host *connect.Host, message *mixmessages.RoundError) (*messages.Ack, error) {
		return nil, nil
	}
	instance.SetRoundErrFunc(mockBroadcast, t)

	for i := 0; i < topology.Len(); i++ {
		nid := topology.GetNodeAtIndex(i)
		params := connect.GetDefaultHostParams()
		params.MaxRetries = 0
		_, err := instance.GetNetwork().AddHost(nid, "0.0.0.0", []byte(testUtil.RegCert), params)
		if err != nil {
			t.Errorf("Failed to add host: %+v", err)
		}
	}

	err := Error(instance, rndErr)
	if err != nil {
		t.Errorf("Failed to error: %+v", err)
	}
	instance.GetNetwork().Shutdown()
}

func TestCrash(t *testing.T) {
	instance, topology := setup(t)
	rndErr := &mixmessages.RoundError{
		Id:     1,
		NodeId: instance.GetID().Marshal(),
		Error:  "",
	}
	mockBroadcast := func(host *connect.Host, message *mixmessages.RoundError) (*messages.Ack, error) {
		return nil, nil
	}
	instance.SetRoundErrFunc(mockBroadcast, t)

	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered in SetGroup(): ", r)
		} else {
			t.Errorf("SetGroup() did not panic when expected while attempting to set the group again")
		}
		instance.GetNetwork().Shutdown()
	}()

	for i := 0; i < topology.Len(); i++ {
		nid := topology.GetNodeAtIndex(i)
		params := connect.GetDefaultHostParams()
		params.MaxRetries = 0
		_, err := instance.GetNetwork().AddHost(nid, "0.0.0.0", []byte(testUtil.RegCert), params)
		if err != nil {
			t.Errorf("Failed to add host: %+v", err)
		}
	}

	err := Crash(instance, rndErr)
	if err != nil {
		t.Errorf("Failed to error: %+v", err)
	}
}

func TestCrash_RID0(t *testing.T) {
	instance, topology := setup(t)
	rndErr := &mixmessages.RoundError{
		Id:     0,
		NodeId: instance.GetID().Marshal(),
		Error:  "",
	}
	mockBroadcast := func(host *connect.Host, message *mixmessages.RoundError) (*messages.Ack, error) {
		t.Error()
		return nil, nil
	}
	instance.SetRoundErrFunc(mockBroadcast, t)

	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered in SetGroup(): ", r)
		} else {
			t.Errorf("SetGroup() did not panic when expected while attempting to set the group again")
		}
		instance.GetNetwork().Shutdown()
	}()

	for i := 0; i < topology.Len(); i++ {
		nid := topology.GetNodeAtIndex(i)
		params := connect.GetDefaultHostParams()
		params.MaxRetries = 0
		_, err := instance.GetNetwork().AddHost(nid, "0.0.0.0", []byte(testUtil.RegCert), params)
		if err != nil {
			t.Errorf("Failed to add host: %+v", err)
		}
	}

	err := Crash(instance, rndErr)
	if err != nil {
		t.Errorf("Failed to error: %+v", err)
	}
}

func TestPrecomputing(t *testing.T) {
	var err error
	instance, topology := setup(t)

	var top [][]byte
	for i := 0; i < topology.Len(); i++ {
		nid := topology.GetNodeAtIndex(i)
		top = append(top, nid.Marshal())
		params := connect.GetDefaultHostParams()
		params.MaxRetries = 0
		_, err = instance.GetNetwork().AddHost(nid, "0.0.0.0", []byte(testUtil.RegCert), params)
		if err != nil {
			t.Errorf("Failed to add host: %+v", err)
		}
	}

	newRoundInfo := &mixmessages.RoundInfo{
		ID:        0,
		Topology:  top,
		BatchSize: 32,
	}

	// Mocking permissioning server signing message
	err = signRoundInfo(newRoundInfo)
	if err != nil {
		t.Errorf("failed to sign round info")
	}

	err = instance.GetConsensus().RoundUpdate(newRoundInfo)
	if err != nil {
		t.Errorf("Failed to updated network instance for new round info: %v", err)
	}

	err = instance.GetCreateRoundQueue().Send(newRoundInfo)
	if err != nil {
		t.Errorf("Failed to send roundInfo: %v", err)
	}

	err = instance.GetResourceQueue().Kill(time.Second)
	if err != nil {
		t.Errorf("Failed to kill resource queue: %+v", err)
	}

	err = Precomputing(instance)
	if err != nil {
		t.Errorf("Failed to precompute: %+v", err)
	}

	_, err = instance.GetRoundManager().GetRound(0)
	if err != nil {
		t.Errorf("A round should have been added to the round manager")
	}
	instance.GetNetwork().Shutdown()
}

func TestPrecomputing_override(t *testing.T) {
	var err error
	instance, topology := setup(t)
	gc := services.NewGraphGenerator(4,
		uint8(runtime.NumCPU()), 1, 0)
	g := graphs.InitErrorGraph(gc)
	th := func(roundID id.Round, instance phase.GenericInstance, getChunk phase.GetChunk, getMessage phase.GetMessage) error {
		return errors.New("Failed intentionally")
	}
	overrides := map[int]phase.Phase{}
	p := phase.New(phase.Definition{
		Graph:               g,
		Type:                phase.PrecompGeneration,
		TransmissionHandler: th,
		Timeout:             30127,
		DoVerification:      false,
	})
	overrides[0] = p
	instance.OverridePhasesAtRound(overrides, 1)

	var top [][]byte
	for i := 0; i < topology.Len(); i++ {
		nid := topology.GetNodeAtIndex(i)
		top = append(top, nid.Marshal())
		params := connect.GetDefaultHostParams()
		params.MaxRetries = 0
		params.AuthEnabled = false
		_, err = instance.GetNetwork().AddHost(nid, "0.0.0.0", []byte(testUtil.RegCert), params)
		if err != nil {
			t.Errorf("Failed to add host: %+v", err)
		}
	}

	newRoundInfo := &mixmessages.RoundInfo{
		ID:        1,
		Topology:  top,
		BatchSize: 32,
	}
	// Mocking permissioning server signing message
	err = signRoundInfo(newRoundInfo)
	if err != nil {
		t.Errorf("failed to sign round info")
	}

	err = instance.GetConsensus().RoundUpdate(newRoundInfo)
	if err != nil {
		t.Errorf("Failed to updated network instance for new round info: %v", err)
	}

	err = instance.GetCreateRoundQueue().Send(newRoundInfo)
	if err != nil {
		t.Errorf("Failed to send roundInfo: %v", err)
	}

	err = instance.GetResourceQueue().Kill(time.Millisecond * 10)
	if err != nil {
		t.Errorf("Failed to kill resource queue: %+v", err)
	}

	err = Precomputing(instance)
	if err != nil {
		t.Errorf("Failed to precompute: %+v", err)
	}

	rnd, _ := instance.GetRoundManager().GetRound(id.Round(1))
	precompPhase, _ := rnd.GetPhase(phase.PrecompGeneration)
	if precompPhase.GetTimeout() != 30127 {
		t.Error("Failed to override phase")
	}
}

// Smoke test: does isRegistered communicate with permissioning server?
func TestIsRegistered(t *testing.T) {
	// Create instance
	instance, pAddr, nAddr, nodeId, cert, key, err := createServerInstance(t)

	if err != nil {
		t.Errorf("Couldn't create instance: %+v", err)
	}

	// Start up permissioning server
	permComms, mockPermissioning, err := startPermissioning(pAddr, nAddr, nodeId, cert, key, t)

	if err != nil {
		t.Errorf("Couldn't create permissioning server: %+v", err)
	}
	defer permComms.Shutdown()

	// Add retrieve permissioning host from instance
	permHost, ok := instance.GetNetwork().GetHost(&id.Permissioning)
	if !ok {
		t.Fatal("Didn't get a permissioning host. Failing now")
	}
	result := isRegistered(instance, permHost)
	const expected = true
	if result != expected {
		t.Errorf("Expected response from mock permissioning to be %v. Got %v instead", expected, result)
	}

	// It should be possible to see this error in the test logs
	expectedErr := errors.New("mock error")
	mockPermissioning.SetDesiredError(expectedErr)
	result = isRegistered(instance, permHost)
	const expectedWhenErr = false
	if result != expectedWhenErr {
		t.Error("isRegistered should return false when permissioning returns an error")
	}
}
