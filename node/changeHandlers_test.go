////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package node

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/comms/connect"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/signature"
	"gitlab.com/elixxir/crypto/signature/rsa"
	"gitlab.com/elixxir/crypto/tls"
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/globals"
	"gitlab.com/elixxir/server/node/receivers"
	"gitlab.com/elixxir/server/server"
	"gitlab.com/elixxir/server/server/measure"
	"gitlab.com/elixxir/server/server/round"
	"gitlab.com/elixxir/server/server/state"
	"gitlab.com/elixxir/server/services"
	"gitlab.com/elixxir/server/testUtil"
	"testing"
)

func setup(t *testing.T) (*server.Instance, *connect.Circuit) {
	var nodeIDs []*id.Node

	//Build IDs
	for i := 0; i < 5; i++ {
		nodIDBytes := make([]byte, id.NodeIdLen)
		nodIDBytes[0] = byte(i + 1)
		nodeID := id.NewNodeFromBytes(nodIDBytes)
		nodeIDs = append(nodeIDs, nodeID)
	}

	//Build the topology
	topology := connect.NewCircuit(nodeIDs)
	gg := services.NewGraphGenerator(4, nil, 1,
		services.AutoOutputSize, 1.0)
	def := server.Definition{
		UserRegistry:    &globals.UserMap{},
		ResourceMonitor: &measure.ResourceMonitor{},
		FullNDF:         testUtil.NDF,
		PartialNDF:      testUtil.NDF,
		GraphGenerator:  gg,
		Gateway: server.GW{
			Address: "0.0.0.0:11420",
		},
		Address: "0.0.0.0:11421",
	}
	def.ID = topology.GetNodeAtIndex(0)

	var instance *server.Instance
	var dummyStates = [current.NUM_STATES]state.Change{
		func(from current.Activity) error { return nil },
		func(from current.Activity) error { return nil },
		func(from current.Activity) error { return nil },
		func(from current.Activity) error { return nil },
		func(from current.Activity) error { return nil },
		func(from current.Activity) error { return nil },
		func(from current.Activity) error { return nil },
		func(from current.Activity) error { return nil },
	}
	m := state.NewTestMachine(dummyStates, current.PRECOMPUTING, t)
	instance, _ := server.CreateServerInstance(&def, receivers.NewImplementation, m, false)
	r := round.NewDummyRoundWithTopology(id.Round(0), 3, topology, t)
	instance.GetRoundManager().AddRound(r)
	_ = instance.Run()
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

/*func TestNotStarted_RoundError(t *testing.T) {
	instance, _ := setup(t)
	err := NotStarted(instance, true)
	if err != nil {
		t.Error(err)
	}
}*/

func TestError(t *testing.T) {
	instance, topology := setup(t)
	rndErr := &mixmessages.RoundError{
		Id:     0,
		NodeId: instance.GetID().String(),
		Error:  "",
	}
	mockBroadcast := func(host *connect.Host, message *mixmessages.RoundError) (*mixmessages.Ack, error) {
		return nil, nil
	}
	instance.SetRoundErrBroadcastFunc(mockBroadcast, t)

	for i := 0; i < topology.Len(); i++ {
		nid := topology.GetNodeAtIndex(i).String()
		_, err := instance.GetNetwork().AddHost(nid, "0.0.0.0", []byte(testUtil.RegCert), true, false)
		if err != nil {
			t.Errorf("Failed to add host: %+v", err)
		}
	}

	go func() {
		time.Sleep(50 * time.Millisecond)
		instance.GetErrChan() <- rndErr
	}()

	err := Error(instance)
	if err != nil {
		t.Errorf("Failed to error: %+v", err)
	}
	instance.GetNetwork().Shutdown()
}

func TestPrecomputing(t *testing.T) {
	instance, topology := setup(t)

	var top []string
	for i := 0; i < topology.Len(); i++ {
		nid := topology.GetNodeAtIndex(i).String()
		top = append(top, nid)
		_, err := instance.GetNetwork().AddHost(nid, "0.0.0.0", []byte(testUtil.RegCert), true, false)
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
	signRoundInfo(newRoundInfo)

	err = instance.GetConsensus().RoundUpdate(newRoundInfo)
	if err != nil {
		t.Errorf("Failed to updated network instance for new round info: %v", err)
	}

	err = instance.GetCreateRoundQueue().Send(newRoundInfo)
	if err != nil {
		t.Errorf("Failed to send roundInfo: %v", err)
	}

	instance.GetResourceQueue().Kill(t)

	err = Precomputing(instance, 3)
	if err != nil {
		t.Errorf("Failed to precompute: %+v", err)
	}

	_, err = instance.GetRoundManager().GetRound(0)
	if err != nil {
		t.Errorf("A round should have been added to the round manager")
	}
	instance.GetNetwork().Shutdown()
}

// Utility function which signs a round info message
func signRoundInfo(ri *mixmessages.RoundInfo) error {
	pk, err := tls.LoadRSAPrivateKey(testUtil.RegPrivKey)
	if err != nil {
		return errors.Errorf("couldn't load privKey: %+v", err)
	}

	ourPrivKey := &rsa.PrivateKey{PrivateKey: *pk}

	signature.Sign(ri, ourPrivKey)
	return nil
}
