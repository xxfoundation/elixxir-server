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
	"gitlab.com/elixxir/server/server/round"
	"sync"
)

// SendFinishRealtime broadcasts the finish realtime message to all other nodes
// It sends all messages concurrently, then waits for all to be done,
// while catching any errors that occurred
func SendFinishRealtime(network *node.NodeComms, roundID id.Round,
	topology *circuit.Circuit, selfID *id.Node) error {

	wg := sync.WaitGroup{}
	errChan := make(chan error)

	nodeID := topology.GetNextNode(selfID)
	for ; !nodeID.Cmp(selfID); nodeID = topology.GetNextNode(nodeID) {
		wg.Add(1)
		go func(dest *id.Node) {
			ack, err := network.SendFinishRealtime(dest,
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
		}(nodeID)
	}

	doneChan := make(chan struct{})
	go func() {
		wg.Wait()
		close(doneChan)
	}()

	var allErr error
LOOP:
	for {
		select {
		case <-doneChan:
			break LOOP
		case err := <-errChan:
			allErr = errors.Errorf("%s\n%s", allErr.Error(), err.Error())
		}
	}
	return allErr
}

// FinishRealtime implements the server gRPC handler for receiving
// a finish realtime message from another node
// It looks up the round by roundID given in the message
// and returns an error if it doesn't exist.
// If it exists, it removes the round from the round manager, effectively
// finishing it
func FinishRealtime(rm *round.Manager, msg *mixmessages.RoundInfo) error {
	rnd, err := rm.GetRound(id.Round(msg.ID))
	if err != nil {
		return err
	}
	rm.DeleteRound(rnd.GetID())
	return nil
}
