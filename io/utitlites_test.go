////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package io

import (
	"fmt"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/comms/connect"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/primitives/circuit"
	"gitlab.com/elixxir/primitives/id"
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
		return errors.New("did you want this")
	}
	mp.indices = append(mp.indices, index)
	return nil
}

func (*MockPhase) EnableVerification() { return }
func (*MockPhase) ConnectToRound(id id.Round, setState phase.Transition,
	getState phase.GetState) {
	return
}
func (*MockPhase) GetGraph() *services.Graph              { return nil }
func (*MockPhase) GetRoundID() id.Round                   { return 0 }
func (*MockPhase) GetType() phase.Type                    { return 0 }
func (*MockPhase) GetState() phase.State                  { return 0 }
func (*MockPhase) AttemptTransitionToQueued() bool        { return false }
func (*MockPhase) TransitionToRunning()                   { return }
func (*MockPhase) UpdateFinalStates() bool                { return false }
func (*MockPhase) GetTransmissionHandler() phase.Transmit { return nil }
func (*MockPhase) GetTimeout() time.Duration              { return 0 }
func (*MockPhase) Cmp(phase.Phase) bool                   { return false }
func (*MockPhase) String() string                         { return "" }

func buildTestNetworkComponents(impls []func() *node.Implementation) ([]*node.NodeComms, *circuit.Circuit) {
	var nodeIDs []*id.Node
	var addrLst []string
	addrFmt := "localhost:500%d"

	//Build IDs and addresses
	for i := 0; i < len(impls); i++ {
		nodeID := &id.Node{}
		nodIDBytes := make([]byte, id.NodeIdLen)
		nodIDBytes[0] = byte(i + 1)
		nodeID.SetBytes(nodIDBytes)
		nodeIDs = append(nodeIDs, nodeID)
		addrLst = append(addrLst, fmt.Sprintf(addrFmt, i))
	}

	//Build the topology
	topology := circuit.New(nodeIDs)

	//build the comms
	var comms []*node.NodeComms

	for i := 0; i < len(impls); i++ {
		var impl *node.Implementation
		if impls[i] != nil {
			impl = impls[i]()
		}
		comms = append(comms,
			node.StartNode(addrLst[i], impl, "", ""))
	}

	//Connect the comms
	for connectFrom := 0; connectFrom < len(impls); connectFrom++ {
		for connectTo := connectFrom + 1; connectTo < len(impls); connectTo++ {
			comms[connectFrom].ConnectToNode(
				topology.GetNodeAtIndex(connectTo),
				&connect.ConnectionInfo{
					Address: addrLst[connectTo],
				})
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
