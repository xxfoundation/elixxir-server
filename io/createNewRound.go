package io

import (
	"fmt"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/primitives/circuit"
	"gitlab.com/elixxir/primitives/id"
	"sync"
)

// TransmitCreateNewRound is run on first node to tell other nodes to create the
// round.  It does not follow the transmitter interface because it is run
// custom through the first node runner.
func TransmitCreateNewRound(network *node.Comms,
	topology *circuit.Circuit, roundID id.Round) error {

	//Every node receives the same roundInfo
	roundInfo := &mixmessages.RoundInfo{ID: uint64(roundID)}

	//Create a waitgroup to track the state
	var wg sync.WaitGroup
	errChan := make(chan error, topology.Len())

	wg.Add(topology.Len())

	//send the message to every node including yourself
	for index := 0; index < topology.Len(); index++ {
		localIndex := index
		go func() {
			// Pull the particular server host object from the commManager
			recipientID := topology.GetNodeAtIndex(localIndex).String()
			recipient, ok := network.Manager.GetHost(recipientID)
			if !ok {
				// If server not found, send through error channel
				errMsg := fmt.Sprintf("Could not find cMix server %s (%d/%d) in comm manager",
					recipientID, localIndex+1, topology.Len())
				errChan <- errors.New(errMsg)
			}
			// Send new round to that particular node
			ack, err := network.SendNewRound(recipient, roundInfo)

			if ack != nil && ack.Error != "" {
				err = errors.Errorf("Remote Server Error: %s", ack.Error)
			}

			if err != nil {
				errChan <- err
			}

			wg.Done()
		}()
	}

	wg.Wait()

	// Return all node comms or ack errors if any
	// as a single error message
	var errs error
	for len(errChan) > 0 {
		err := <-errChan
		if errs != nil {
			errs = errors.Wrap(errs, err.Error())
		} else {
			errs = err
		}
	}

	return errs
}
