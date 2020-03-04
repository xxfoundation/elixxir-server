////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Package io finishRealtime.go handles the endpoints and helper functions for
// receiving and sending the finish realtime message between cMix nodes.

package io

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/comms/connect"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/server"
	"gitlab.com/elixxir/server/server/measure"
	"gitlab.com/elixxir/server/server/phase"
	"gitlab.com/elixxir/server/services"
	"sync"
)

// TransmitFinishRealtime broadcasts the finish realtime message to all other nodes
// It sends all messages concurrently, then waits for all to be done,
// while catching any errors that occurred
func TransmitFinishRealtime(network *node.Comms, batchSize uint32,
	roundID id.Round, phaseTy phase.Type, getChunk phase.GetChunk,
	getMessage phase.GetMessage, topology *connect.Circuit,
	nodeID *id.Node, lastNode *server.Instance, measureFunc phase.Measure) error {

	var wg sync.WaitGroup
	errChan := make(chan error, topology.Len())

	// need to send batch to gateway reception handler & trigger permissioning polling handler
	// Update to completed state, and if we are last node send the proper
	// Make a channel to push things into
	complete := server.CompletedRound{
		RoundID:    roundID,
		Receiver:   chunkChan,
		GetMessage: getMessage,
	}

	lastNode.SendCompletedBatchQueue(complete)

	for chunk, finish := getChunk(); finish; chunk, finish = getChunk() {
		chunkChan <- chunk
	}

	close(chunkChan)

	if measureFunc != nil {
		measureFunc(measure.TagTransmitLastSlot)
	}

	// signal to all nodes except the first node that the round has been
	// completed. Skip the first node and do it after to ensure all measurements
	// are stored before it polls for the measurement data
	for index := 0; index < topology.Len(); index++ {
		localIndex := index
		wg.Add(1)
		go func() {
			// Pull the particular server host object from the commManager
			recipient := topology.GetHostAtIndex(localIndex)
			// Send the message to that particular node
			ack, err := network.SendFinishRealtime(recipient,
				&mixmessages.RoundInfo{
					ID: uint64(roundID),
				})

			if ack != nil && ack.Error != "" {
				err = errors.Errorf("Remote Server Error: %s", ack.Error)
			}

			if err != nil {
				errChan <- err
			}
			wg.Done()
		}()

	}

	// Wait for all responses
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
