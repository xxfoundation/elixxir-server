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
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/primitives/circuit"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/server/phase"
	"gitlab.com/elixxir/server/server/round"
	"sync"
)

// TransmitFinishRealtime broadcasts the finish realtime message to all other nodes
// It sends all messages concurrently, then waits for all to be done,
// while catching any errors that occurred
func TransmitFinishRealtime(network *node.NodeComms, batchSize uint32,
	roundID id.Round, phaseTy phase.Type, getChunk phase.GetChunk,
	getMessage phase.GetMessage, topology *circuit.Circuit,
	nodeID *id.Node) error {

	var wg sync.WaitGroup
	errChan := make(chan error, topology.Len()-1)

	for index := 1; index < topology.Len(); index++ {
		localIndex := index
		wg.Add(1)
		go func() {
			recipient := topology.GetNodeAtIndex(localIndex)

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

	if errs != nil {
		return errs
	}

	// If we got here, there weren't errors, so let's send to the first node
	// so the round will be finished
	recipient := topology.GetNodeAtIndex(0)
	ack, err := network.SendFinishRealtime(recipient,
		&mixmessages.RoundInfo{
			ID: uint64(roundID),
		})
	if err != nil {
		return err
	} else if ack != nil && ack.Error != "" {
		return errors.Errorf("Remote error: %v", ack.Error)
	} else {
		return nil
	}
}

// FinishRealtime implements the server gRPC handler for receiving
// a finish realtime message from another node
// It looks up the round by roundID given in the message
// and returns an error if it doesn't exist.
// If it exists, it removes the round from the round manager, effectively
// finishing it
func FinishRealtime(rm *round.Manager, roundID id.Round) error {
	rnd, err := rm.GetRound(roundID)
	if err != nil {
		return err
	}
	rm.DeleteRound(rnd.GetID())
	return nil
}
