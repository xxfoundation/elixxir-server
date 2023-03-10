////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package io

// transmitPostPrecompResult.go contains the logic for transmitting a precompResult comm

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/server/internal"
	"gitlab.com/elixxir/server/internal/measure"
	"gitlab.com/elixxir/server/internal/phase"
	"gitlab.com/elixxir/server/internal/round"
	"gitlab.com/xx_network/primitives/id"
	"sync"
)

// TransmitPrecompResult: The last node transmits the precomputation to all
// nodes but the first, then the first node, after precomp strip
func TransmitPrecompResult(roundID id.Round, serverInstance phase.GenericInstance, getChunk phase.GetChunk,
	getMessage phase.GetMessage) error {

	var wg sync.WaitGroup
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

	errChan := make(chan error, topology.Len()-1)
	// Build the message containing the precomputations

	slots := make([]*mixmessages.Slot, r.GetBatchSize())

	// For each message chunk (slot), fill the slots buffer
	// Note that this will panic if there are more slots than batchSize
	// (shouldn't be possible?)
	for chunk, finish := getChunk(); finish; chunk, finish = getChunk() {
		for i := chunk.Begin(); i < chunk.End(); i++ {
			msg := getMessage(i)
			slots[i] = msg
		}
	}

	measureFunc := r.GetCurrentPhase().Measure
	if measureFunc != nil {
		measureFunc(measure.TagTransmitLastSlot)
	}

	//store the precomp result internally to be used in realtime
	err = PostPrecompResult(r.GetBuffer(), instance.GetNetworkStatus().GetCmixGroup(), slots)
	if err != nil {
		return errors.WithMessage(err, "Failed to finish precomputation")
	}

	// Send to all nodes
	for i := 0; i < topology.Len(); i++ {
		wg.Add(1)
		go func(index int) {
			var err error
			// Pull the particular server host object from the commManager
			recipient := topology.GetHostAtIndex(index)

			ack, err := instance.GetNetwork().SendPostPrecompResult(
				recipient, uint64(roundID), uint32(len(slots)))
			if err != nil {
				errChan <- errors.Wrapf(err, "")
			}
			if ack != nil && ack.Error != "" {
				errChan <- errors.Errorf("Remote error: %v", ack.Error)
			}

			wg.Done()
		}(i)
	}
	wg.Wait()

	// Return any errors at this point
	var errs error
	for len(errChan) > 0 {
		err := <-errChan
		if err != nil {
			if errs != nil {
				errs = errors.Wrap(errs, err.Error())
			} else {
				errs = err
			}
		}
	}
	if errs != nil {
		return errs
	}
	return nil
}

func PostPrecompResult(r *round.Buffer, grp *cyclic.Group,
	slots []*mixmessages.Slot) error {
	batchSize := r.GetBatchSize()
	if batchSize != uint32(len(slots)) {
		return errors.New("PostPrecompResult: The number of slots we got" +
			" wasn't equal to the number of slots in the buffer")
	}
	for i := uint32(0); i < uint32(len(slots)); i++ {
		PayloadAPrecomputation := r.PayloadAPrecomputation.Get(i)
		PayloadBPrecomputation := r.PayloadBPrecomputation.Get(i)
		grp.SetBytes(PayloadAPrecomputation, slots[i].EncryptedPayloadAKeys)
		grp.SetBytes(PayloadBPrecomputation, slots[i].EncryptedPayloadBKeys)
	}

	return nil
}
