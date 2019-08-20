////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package io

import (
	"fmt"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/primitives/circuit"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/server/measure"
	"gitlab.com/elixxir/server/server/phase"
	"gitlab.com/elixxir/server/services"
	"time"
)

type MockPhase struct {
	chunks  []services.Chunk
	indices []uint32
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

func (*MockPhase) EnableVerification() { return }
func (*MockPhase) ConnectToRound(id id.Round, setState phase.Transition,
	getState phase.GetState) {
	return
}
func (*MockPhase) GetGraph() *services.Graph { return nil }
func (*MockPhase) GetRoundID() id.Round      { return 0 }
func (*MockPhase) GetType() phase.Type       { return 0 }
func (*MockPhase) GetState() phase.State     { return 0 }
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

func buildTestNetworkComponents(impls []*node.Implementation,
	portStart int) ([]*node.NodeComms, *circuit.Circuit) {
	var nodeIDs []*id.Node
	var addrLst []string
	addrFmt := "localhost:3%03d"

	//Build IDs and addresses
	for i := 0; i < len(impls); i++ {
		nodIDBytes := make([]byte, id.NodeIdLen)
		nodIDBytes[0] = byte(i + 1)
		nodeID := id.NewNodeFromBytes(nodIDBytes)
		nodeIDs = append(nodeIDs, nodeID)
		addrLst = append(addrLst, fmt.Sprintf(addrFmt, i+portStart))
	}

	//Build the topology
	topology := circuit.New(nodeIDs)

	//build the comms
	var comms []*node.NodeComms

	for index, impl := range impls {
		comms = append(comms,
			node.StartNode(addrLst[index], impl, nil, nil))
	}

	//Connect the comms
	for connectFrom := 0; connectFrom < len(impls); connectFrom++ {
		for connectTo := 0; connectTo < len(impls); connectTo++ {
			comms[connectFrom].ConnectToNode(
				topology.GetNodeAtIndex(connectTo),
				addrLst[connectTo], nil)
		}
	}

	//Return comms and topology
	return comms, topology
}

func Shutdown(comms []*node.NodeComms) {
	for _, comm := range comms {
		comm.Shutdown()
	}
}
