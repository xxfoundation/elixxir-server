///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

// Package io transmitFinishRealtime.go handles the endpoints and helper functions for
// receiving and sending the finish realtime message between cMix nodes.

package io

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/server/internal"
	"gitlab.com/elixxir/server/internal/measure"
	"gitlab.com/elixxir/server/internal/phase"
	"gitlab.com/elixxir/server/internal/round"
	"gitlab.com/xx_network/primitives/id"
	"sync"
)

// TransmitFinishRealtime broadcasts the finish realtime message to all other nodes
// It sends all messages concurrently, then waits for all to be done,
// while catching any errors that occurred
func TransmitFinishRealtime(roundID id.Round, serverInstance phase.GenericInstance, getChunk phase.GetChunk, getMessage phase.GetMessage) error {
	instance, ok := serverInstance.(*internal.Instance)
	if !ok {
		return errors.Errorf("Invalid server instance passed in")
	}

	//get the round so you can get its batch size
	r, err := instance.GetRoundManager().GetRound(roundID)
	if err != nil {
		return errors.Errorf("Could not retrieve round %d from manager  %s", roundID, err)
	}

	topology := r.GetTopology()

	// Form completed round object & push to gateway handler
	complete := &round.CompletedRound{
		RoundID: roundID,
		Round:   make([]*mixmessages.Slot, r.GetBatchSize()),
	}

	// For each message chunk (slot), fill the slots buffer
	// Note that this will panic if there are more slots than batchSize
	// (shouldn't be possible?)
	for chunk, finish := getChunk(); finish; chunk, finish = getChunk() {
		for i := chunk.Begin(); i < chunk.End(); i++ {
			msg := getMessage(i)
			complete.Round[i] = msg
		}
	}

	measureFunc := r.GetCurrentPhase().Measure
	if measureFunc != nil {
		measureFunc(measure.TagTransmitLastSlot)
	}

	// signal to all team members that the round has been
	// completed by sending the completed batch.
	var wg sync.WaitGroup
	errChan := make(chan error, topology.Len())
	for index := 0; index < topology.Len(); index++ {
		localIndex := index
		wg.Add(1)
		go func() {
			// Pull the particular server host object from the commManager
			recipient := topology.GetHostAtIndex(localIndex)
			// Send the message to that particular node
			ack, err := instance.GetNetwork().SendFinishRealtime(recipient,
				&mixmessages.RoundInfo{ID: uint64(roundID)},
				&mixmessages.CompletedBatch{Slots: complete.Round},
			)

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
	numerrors := 0
	for len(errChan) > 0 {
		numerrors++
		err := <-errChan
		if errs != nil {
			errs = errors.Wrap(errs, err.Error())
		} else {
			errs = err
		}
	}

	if numerrors>0{
		jww.ERROR.Printf("Streaming batch to other nodes failed in " +
			"round %d for %d/%d nodes: %+v", roundID, numerrors, topology.Len(),
			errs)
	}

	return nil
}
