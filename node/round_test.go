////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package node

import (
	"gitlab.com/elixxir/server/services"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/id"
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

	// Dummy instance to prevent segfault
	instance, _, _, _, _, _, _ := createServerInstance(t)

	phases, responses := NewRoundComponents(gc, topology, nodeID, instance, 2*time.Second, nil, false, 0)

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
	expectedMiddleNodeResponses := 9

	gc := services.NewGraphGenerator(4, 1,
		services.AutoOutputSize, 1.0)

	topology := buildMockTopology(3, t)

	nodeID := topology.GetNodeAtIndex(1)

	// Dummy instance to prevent segfault
	instance, _, _, _, _, _, _ := createServerInstance(t)

	phases, responses := NewRoundComponents(gc, topology, nodeID, instance, 2*time.Second, nil, false, 0)

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
	expectedLastNodeResponses := 9

	gc := services.NewGraphGenerator(4, 1,
		services.AutoOutputSize, 1.0)

	topology := buildMockTopology(3, t)

	nodeID := topology.GetNodeAtIndex(2)

	// Dummy instance to prevent segfault
	instance, _, _, _, _, _, _ := createServerInstance(t)

	phases, responses := NewRoundComponents(gc, topology, nodeID, instance, 2*time.Second, nil, false, 0)

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

	// Dummy instance to prevent segfault
	instance, _, _, _, _, _, _ := createServerInstance(t)

	phases, responses := NewRoundComponents(gc, topology, nodeID, instance, 2*time.Second, nil, true, 0)

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
	expectedMiddleNodeResponses := 9

	gc := services.NewGraphGenerator(4, 1,
		services.AutoOutputSize, 1.0)

	topology := buildMockTopology(3, t)

	nodeID := topology.GetNodeAtIndex(1)

	// Dummy instance to prevent segfault
	instance, _, _, _, _, _, _ := createServerInstance(t)

	phases, responses := NewRoundComponents(gc, topology, nodeID, instance, 2*time.Second, nil, true, 0)

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
	expectedLastNodeResponses := 9

	gc := services.NewGraphGenerator(4, 1,
		services.AutoOutputSize, 1.0)

	topology := buildMockTopology(3, t)

	nodeID := topology.GetNodeAtIndex(2)

	// Dummy instance to prevent segfault
	instance, _, _, _, _, _, _ := createServerInstance(t)

	phases, responses := NewRoundComponents(gc, topology, nodeID, instance, 2*time.Second, nil, true, 0)

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
