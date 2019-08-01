package io

import (
	"fmt"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/server"
	"testing"
	"time"
)

var receivedNewRound = make(chan *mixmessages.RoundInfo, 100)

func MockCreateNewRoundImplementation() *node.Implementation {
	impl := node.NewImplementation()
	impl.Functions.CreateNewRound = func(message *mixmessages.RoundInfo) error {
		receivedNewRound <- message
		return nil
	}
	return impl
}

// Test that TransmitFinishRealtime correctly broadcasts message
// to all other nodes
func TestTransmitCreateNewRound(t *testing.T) {
	//Setup the network
	numNodes := 5
	numRecv := 0
	// init every node including yourself because the first node uses the comm
	// to create the new round for simplicity
	comms, topology := buildTestNetworkComponents(
		[]*node.Implementation{
			MockCreateNewRoundImplementation(),
			MockCreateNewRoundImplementation(),
			MockCreateNewRoundImplementation(),
			MockCreateNewRoundImplementation(),
			MockCreateNewRoundImplementation()}, 2000)
	defer Shutdown(comms)

	rndID := id.Round(42)

	ln := server.LastNode{}
	ln.Initialize()

	err := TransmitCreateNewRound(comms[0], topology, rndID)

	if err != nil {
		t.Errorf("TransmitFinishRealtime: Unexpected error: %+v", err)
	}

Loop:
	for {
		select {
		case msg := <-receivedNewRound:
			if id.Round(msg.ID) != rndID {
				t.Errorf("TransmitFinishRealtime: Incorrect round ID"+
					"Expected: %v, Received: %v", rndID, msg.ID)
			}
			numRecv++
			if numRecv == numNodes {
				break Loop
			}
		case <-time.After(5 * time.Second):
			t.Errorf("Test timed out!")
			break Loop
		}
	}
}

func MockCreateNewRoundImplementation_Error() *node.Implementation {
	impl := node.NewImplementation()
	impl.Functions.CreateNewRound = func(message *mixmessages.RoundInfo) error {
		return errors.New("Test error")
	}
	return impl
}

//Tests that the error handeling code works properly
func TestTransmitCreateNewRound_Error(t *testing.T) {
	//Setup the network
	comms, topology := buildTestNetworkComponents(
		[]*node.Implementation{
			MockCreateNewRoundImplementation_Error(),
			MockCreateNewRoundImplementation_Error(),
			MockCreateNewRoundImplementation_Error(),
			MockCreateNewRoundImplementation_Error(),
			MockCreateNewRoundImplementation_Error()}, 2000)
	defer Shutdown(comms)

	rndID := id.Round(42)

	ln := server.LastNode{}
	ln.Initialize()

	err := TransmitCreateNewRound(comms[0], topology, rndID)

	if err == nil {
		t.Error("SendFinishRealtime: error did not occur when provoked")
	}

	fmt.Println(err.Error())

}
