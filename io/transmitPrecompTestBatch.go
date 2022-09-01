////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package io

import (
	"github.com/pkg/errors"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/server/internal"
	"gitlab.com/elixxir/server/internal/phase"
	"gitlab.com/elixxir/server/internal/round"
	"gitlab.com/xx_network/primitives/id"
	"sync"
)

// TransmitPrecompTestBatch is a streaming transmitter which transmits a
// test batch of random data from the last node in order to verify the data
// can be sent over the connection because a similar set of data will be
// sent on the last leg of realtime. It is called in the Precomputing change handler.
func TransmitPrecompTestBatch(roundId id.Round, serverInstance phase.GenericInstance) error {
	instance, ok := serverInstance.(*internal.Instance)
	if !ok {
		return errors.Errorf("Invalid server instance passed in")
	}

	//get the round so you can get its batch size
	r, err := instance.GetRoundManager().GetRound(roundId)
	if err != nil {
		return errors.Errorf("Could not retrieve round %d from manager  %s", roundId, err)
	}

	topology := r.GetTopology()

	primesize := instance.GetNetworkStatus().GetCmixGroup().GetP().ByteLen()

	slots, err := makePrecompTestBatch(instance, r, primesize)
	if err != nil {
		return errors.WithMessage(err, "Failed to construct random test batch")
	}

	// Send mock batch to all nodes in team
	var wg sync.WaitGroup
	errChan := make(chan error, topology.Len())
	for index := 0; index < topology.Len(); index++ {
		wg.Add(1)
		go func(index int) {
			// Pull the particular server host object from the commManager
			recipient := topology.GetHostAtIndex(index)
			// Send the message to that particular node
			err := instance.GetNetwork().StreamPrecompTestBatch(recipient,
				&pb.RoundInfo{ID: uint64(roundId)},
				&pb.CompletedBatch{Slots: slots},
			)

			if err != nil {
				errChan <- err
			}
			wg.Done()
		}(index)
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

// Helper function which generates a batch with slots containing random data.
// This mocks the completed batch sent over realtime.
func makePrecompTestBatch(instance *internal.Instance,
	r *round.Round, primeSize int) ([]*pb.Slot, error) {
	rng := instance.GetRngStreamGen().GetStream()
	slots := make([]*pb.Slot, 0, r.GetBatchSize())
	for i := 0; i < int(r.GetBatchSize()); i++ {
		slot := &pb.Slot{
			Index:    uint32(i),
			PayloadA: make([]byte, primeSize),
			PayloadB: make([]byte, primeSize),
		}
		_, errA := rng.Read(slot.PayloadA)
		_, errB := rng.Read(slot.PayloadB)

		if errA != nil {
			return nil, errors.WithMessagef(errA, "Failed to generate random data for slot %d", i)
		} else if errB != nil {
			return nil, errors.WithMessagef(errB, "Failed to generate random data for slot %d", i)
		}
		slots = append(slots, slot)
	}
	rng.Close()

	return slots, nil
}
