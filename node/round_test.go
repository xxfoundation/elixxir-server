package node

import (
	"gitlab.com/elixxir/server/services"
	"testing"
)

const expectedNumPhases = 7

func TestNewRoundComponents_FirstNode(t *testing.T) {
	expectedFirstNodeResponses := 7

	gc := services.NewGraphGenerator(4, nil, 1,
		services.AutoOutputSize, 1.0)

	topology := buildMockTopology(3)

	nodeID := topology.GetNodeAtIndex(0)

	phases, responses := NewRoundComponents(gc, topology, nodeID, nil,
		100)

	if len(phases) != expectedNumPhases {
		t.Errorf("NewRoundComponents: incorrect number for phases for "+
			"First Node; Expected: %v, Recieved: %v", expectedNumPhases,
			len(phases))
	}

	if len(responses) != expectedFirstNodeResponses {
		t.Errorf("NewRoundComponents: incorrect number for responses "+
			"for First Node; Expected: %v, Recieved: %v",
			expectedFirstNodeResponses, len(responses))
	}

}

func TestNewRoundComponents_MiddleNode(t *testing.T) {
	expectedMiddleNodeResponses := 10

	gc := services.NewGraphGenerator(4, nil, 1,
		services.AutoOutputSize, 1.0)

	topology := buildMockTopology(3)

	nodeID := topology.GetNodeAtIndex(1)

	phases, responses := NewRoundComponents(gc, topology, nodeID, nil,
		100)

	if len(phases) != expectedNumPhases {
		t.Errorf("NewRoundComponents: incorrect number for phases for "+
			"Middle Node; Expected: %v, Recieved: %v", expectedNumPhases,
			len(phases))
	}

	if len(responses) != expectedMiddleNodeResponses {
		t.Errorf("NewRoundComponents: incorrect number for responces "+
			"for Middle Node; Expected: %v, Recieved: %v",
			expectedMiddleNodeResponses, len(responses))
	}
}

func TestNewRoundComponents_LastNode(t *testing.T) {
	expectedLastNodeResponses := 10

	gc := services.NewGraphGenerator(4, nil, 1,
		services.AutoOutputSize, 1.0)

	topology := buildMockTopology(3)

	nodeID := topology.GetNodeAtIndex(2)

	phases, responses := NewRoundComponents(gc, topology, nodeID, nil,
		100)

	if len(phases) != expectedNumPhases {
		t.Errorf("NewRoundComponents: incorrect number for phases for "+
			"Last Node; Expected: %v, Recieved: %v", expectedNumPhases,
			len(phases))
	}

	if len(responses) != expectedLastNodeResponses {
		t.Errorf("NewRoundComponents: incorrect number for responces "+
			"for Last Node; Expected: %v, Recieved: %v",
			expectedLastNodeResponses, len(responses))
	}
}
