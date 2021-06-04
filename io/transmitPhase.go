///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

// Package io transmitPhase.go handles the endpoints and helper functions for
// receiving and sending batches of cMix messages through phases.

package io

import (
	"fmt"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/server/internal"
	"gitlab.com/elixxir/server/internal/measure"
	"gitlab.com/elixxir/server/internal/phase"
	"gitlab.com/elixxir/server/services"
	"gitlab.com/xx_network/primitives/id"
	"strings"
	"sync"
	"time"
)

const shotgunSize = 16

// TransmitPhase sends a cMix Batch of messages to the provided Node.
func TransmitPhase(roundID id.Round, serverInstance phase.GenericInstance, getChunk phase.GetChunk,
	getMessage phase.GetMessage) error {

	instance, ok := serverInstance.(*internal.Instance)
	if !ok {
		return errors.Errorf("Invalid server instance passed in")
	}

	//get the round so you can get its batch size
	r, err := instance.GetRoundManager().GetRound(roundID)
	if err != nil {
		return errors.Errorf("Received completed batch for round %v that doesn't exist: %s", roundID, err)
	}
	rType := r.GetCurrentPhaseType()

	topology := r.GetTopology()
	nodeId := instance.GetID()

	//fixme: for precompShare r.getBatchsize is not the correct value of the batch size and
	// results in nil slots being sent out. Possibily need to update on the fly
	//  alternatively, just use this and it works. Does the reviewer have any thoughts?
	//batchSize := r.GetCurrentPhase().GetGraph().GetBatchSize()
	currentPhase := r.GetCurrentPhase()

	// Pull the particular server host object from the commManager
	recipientID := topology.GetNextNode(nodeId)
	nextNodeIndex := topology.GetNodeLocation(recipientID)
	recipient := topology.GetHostAtIndex(nextNodeIndex)

	// Create the message structure to send the messages
	batch := &mixmessages.Batch{
		Round: &mixmessages.RoundInfo{
			ID: uint64(roundID),
		},
		FromPhase: int32(r.GetCurrentPhaseType()),
		Slots:     make([]*mixmessages.Slot, 0, shotgunSize),
	}

	// For each message chunk (slot), fill the slots buffer
	// Note that this will panic if there are more slots than batchSize
	// (shouldn't be possible?)
	cnt := 0
	wg := sync.WaitGroup{}
	start := time.Now()
	for chunk, finish := getChunk(); finish; chunk, finish = getChunk() {
		for i := chunk.Begin(); i < chunk.End(); i++ {
			msg := getMessage(i)
			batch.Slots = append(batch.Slots, msg)
			cnt++
			if len(batch.Slots) == shotgunSize {
				localBatch := batch
				wg.Add(1)
				go func() {
					// Make sure the comm doesn't return an Ack with an error message
					ack, err := instance.GetNetwork().SendPostPhase(recipient, localBatch)
					if ack != nil && ack.Error != "" {
						err = errors.Errorf("Remote Server Error: %s", ack.Error)
					}
					if err != nil {
						jww.ERROR.Printf("Error on phase transmit: %s", err.Error())
					}
					wg.Done()

				}()
				batch = &mixmessages.Batch{
					Round: &mixmessages.RoundInfo{
						ID: uint64(roundID),
					},
					FromPhase: int32(r.GetCurrentPhaseType()),
					Slots:     make([]*mixmessages.Slot, 0, shotgunSize),
				}
			}
		}
	}

	if len(batch.Slots) > 0 {
		localBatch := batch
		wg.Add(1)
		go func() {
			// Make sure the comm doesn't return an Ack with an error message
			ack, err := instance.GetNetwork().SendPostPhase(recipient, localBatch)
			if ack != nil && ack.Error != "" {
				err = errors.Errorf("Remote Server Error: %s", ack.Error)
			}
			if err != nil {
				jww.ERROR.Printf("Error on phase transmit: %s", err.Error())
			}
			wg.Done()

		}()
	}

	wg.Wait()
	end := time.Now()

	localServer := instance.GetNetwork().String()
	port := strings.Split(localServer, ":")[1]
	addr := fmt.Sprintf("%s:%s", nodeId, port)
	name := services.NameStringer(addr, topology.GetNodeLocation(nodeId), topology.Len())

	measureFunc := r.GetCurrentPhase().Measure
	if measureFunc != nil {
		measureFunc(measure.TagTransmitLastSlot)
	}
	jww.INFO.Printf("[%s]: RID %d TransmitPhase FOR \"%s\" COMPLETE/SEND",
		name, roundID, rType)

	jww.INFO.Printf("\tbwLogging shotgun: Round %d, "+
		"transmitted phase: %s, "+
		"from: %s, to: %s, "+
		"started: %v, "+
		"ended: %v, "+
		"duration: %d",
		roundID, currentPhase.GetType(),
		instance.GetID(), recipientID,
		start, end, end.Sub(start).Milliseconds())

	return err
}

// PostPhase implements the server gRPC handler for posting a
// phase from another node
func PostPhase(p phase.Phase, batch *mixmessages.Batch) error {
	// Send a chunk per slot
	for _, message := range batch.Slots {
		jww.INFO.Println(message.Index)
		err := p.Input(message.Index, message)
		if err != nil {
			return errors.Errorf("Error on Round %v, phase \"%s\" "+
				"slot %d, contents: %v: %v", batch.Round.ID, phase.Type(batch.FromPhase),
				message.Index, message, err)
		}
		chunk := services.NewChunk(message.Index, message.Index+1)
		p.Send(chunk)
	}

	return nil
}
