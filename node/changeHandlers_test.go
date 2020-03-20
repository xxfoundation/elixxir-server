////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package node

import (
	"gitlab.com/elixxir/comms/connect"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/globals"
	"gitlab.com/elixxir/server/node/receivers"
	"gitlab.com/elixxir/server/server"
	"gitlab.com/elixxir/server/server/measure"
	"gitlab.com/elixxir/server/server/state"
	"gitlab.com/elixxir/server/services"
	"gitlab.com/elixxir/server/testUtil"
	"testing"
	"time"
)

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

func TestPrecomputing(t *testing.T) {

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
	}
	def.ID = topology.GetNodeAtIndex(0)

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
	_ = instance.Run()
	var top []string
	for i := 0; i < topology.Len(); i++ {
		nid := topology.GetNodeAtIndex(i).String()
		top = append(top, nid)
		_, err := instance.GetNetwork().AddHost(nid, "0.0.0.0", []byte(testUtil.RegCert), true, false)
		if err != nil {
			t.Errorf("Failed to add host: %+v", err)
		}
	}
	go func() {
		time.Sleep(time.Second)
		instance.GetCreateRoundQueue() <- &mixmessages.RoundInfo{
			ID:        0,
			Topology:  top,
			BatchSize: 32,
		}
	}()
	instance.GetResourceQueue().Kill(t)

	err := Precomputing(instance, 3)
	if err != nil {
		t.Errorf("Failed to precompute: %+v", err)
	}

	_, err = instance.GetRoundManager().GetRound(0)
	if err != nil {
		t.Errorf("A round should have been added to the round manager")
	}
}
