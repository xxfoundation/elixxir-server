///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package node

import (
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/services"
	"gitlab.com/xx_network/comms/connect"
	"testing"
	"time"
)

const expectedNumPhases = 7

func TestNewRoundComponents_FirstNode(t *testing.T) {
	expectedFirstNodeResponses := 7

	gc := services.NewGraphGenerator(4, 1,
		services.AutoOutputSize, 1.0)

	topology := buildMockTopology(3, t)

	nodeID := topology.GetNodeAtIndex(0)

	phases, responses := NewRoundComponents(gc, topology, nodeID, nil,
		100, 2*time.Second, nil, false)

	if len(phases) != expectedNumPhases {
		t.Errorf("NewRoundComponents: incorrect number for phases for "+
			"First Node; Expected: %v, Received: %v", expectedNumPhases,
			len(phases))
	}

	if len(responses) != expectedFirstNodeResponses {
		t.Errorf("NewRoundComponents: incorrect number for responses "+
			"for First Node; Expected: %v, Received: %v",
			expectedFirstNodeResponses, len(responses))
	}

}

func TestNewRoundComponents_MiddleNode(t *testing.T) {
	expectedMiddleNodeResponses := 10

	gc := services.NewGraphGenerator(4, 1,
		services.AutoOutputSize, 1.0)

	topology := buildMockTopology(3, t)

	nodeID := topology.GetNodeAtIndex(1)

	phases, responses := NewRoundComponents(gc, topology, nodeID, nil,
		100, 2*time.Second, nil, false)

	if len(phases) != expectedNumPhases {
		t.Errorf("NewRoundComponents: incorrect number for phases for "+
			"Middle Node; Expected: %v, Received: %v", expectedNumPhases,
			len(phases))
	}

	if len(responses) != expectedMiddleNodeResponses {
		t.Errorf("NewRoundComponents: incorrect number for responces "+
			"for Middle Node; Expected: %v, Received: %v",
			expectedMiddleNodeResponses, len(responses))
	}
}

func TestNewRoundComponents_LastNode(t *testing.T) {
	expectedLastNodeResponses := 10

	gc := services.NewGraphGenerator(4, 1,
		services.AutoOutputSize, 1.0)

	topology := buildMockTopology(3, t)

	nodeID := topology.GetNodeAtIndex(2)

	phases, responses := NewRoundComponents(gc, topology, nodeID, nil,
		100, 2*time.Second, nil, false)

	if len(phases) != expectedNumPhases {
		t.Errorf("NewRoundComponents: incorrect number for phases for "+
			"Last Node; Expected: %v, Received: %v", expectedNumPhases,
			len(phases))
	}

	if len(responses) != expectedLastNodeResponses {
		t.Errorf("NewRoundComponents: incorrect number for responces "+
			"for Last Node; Expected: %v, Received: %v",
			expectedLastNodeResponses, len(responses))
	}
}

func TestNewRoundComponents_FirstNode_Streaming(t *testing.T) {
	expectedFirstNodeResponses := 7

	gc := services.NewGraphGenerator(4, 1,
		services.AutoOutputSize, 1.0)

	topology := buildMockTopology(3, t)

	nodeID := topology.GetNodeAtIndex(0)

	phases, responses := NewRoundComponents(gc, topology, nodeID, nil,
		100, 2*time.Second, nil, true)

	if len(phases) != expectedNumPhases {
		t.Errorf("NewRoundComponents: incorrect number for phases for "+
			"First Node; Expected: %v, Received: %v", expectedNumPhases,
			len(phases))
	}

	if len(responses) != expectedFirstNodeResponses {
		t.Errorf("NewRoundComponents: incorrect number for responses "+
			"for First Node; Expected: %v, Received: %v",
			expectedFirstNodeResponses, len(responses))
	}

}

func TestNewRoundComponents_MiddleNode_Streaming(t *testing.T) {
	expectedMiddleNodeResponses := 10

	gc := services.NewGraphGenerator(4, 1,
		services.AutoOutputSize, 1.0)

	topology := buildMockTopology(3, t)

	nodeID := topology.GetNodeAtIndex(1)

	phases, responses := NewRoundComponents(gc, topology, nodeID, nil,
		100, 2*time.Second, nil, true)

	if len(phases) != expectedNumPhases {
		t.Errorf("NewRoundComponents: incorrect number for phases for "+
			"Middle Node; Expected: %v, Received: %v", expectedNumPhases,
			len(phases))
	}

	if len(responses) != expectedMiddleNodeResponses {
		t.Errorf("NewRoundComponents: incorrect number for responces "+
			"for Middle Node; Expected: %v, Received: %v",
			expectedMiddleNodeResponses, len(responses))
	}
}

func TestNewRoundComponents_LastNode_Streaming(t *testing.T) {
	expectedLastNodeResponses := 10

	gc := services.NewGraphGenerator(4, 1,
		services.AutoOutputSize, 1.0)

	topology := buildMockTopology(3, t)

	nodeID := topology.GetNodeAtIndex(2)

	phases, responses := NewRoundComponents(gc, topology, nodeID, nil,
		100, 2*time.Second, nil, true)

	if len(phases) != expectedNumPhases {
		t.Errorf("NewRoundComponents: incorrect number for phases for "+
			"Last Node; Expected: %v, Received: %v", expectedNumPhases,
			len(phases))
	}

	if len(responses) != expectedLastNodeResponses {
		t.Errorf("NewRoundComponents: incorrect number for responces "+
			"for Last Node; Expected: %v, Received: %v",
			expectedLastNodeResponses, len(responses))
	}
}

// Builds a list of node IDs for testing
func buildMockTopology(numNodes int, t *testing.T) *connect.Circuit {
	var nodeIDs []*id.ID

	// Build IDs
	for i := 0; i < numNodes; i++ {
		nodIDBytes := make([]byte, id.ArrIDLen)
		nodIDBytes[0] = byte(i + 1)
		nodeID := id.NewIdFromBytes(nodIDBytes, t)
		nodeIDs = append(nodeIDs, nodeID)
	}

	// Build the topology
	return connect.NewCircuit(nodeIDs)
}
