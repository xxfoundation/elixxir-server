///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package io

// transmitPhasestream.go contains the logic for streaming a phase comm

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/server/internal"
	"gitlab.com/elixxir/server/internal/measure"
	"gitlab.com/elixxir/server/internal/phase"
	"gitlab.com/elixxir/server/services"
	"gitlab.com/xx_network/comms/messages"
	"gitlab.com/xx_network/primitives/id"
	"google.golang.org/protobuf/proto"
	"io"
	"strings"
	"sync"
	"time"
)

const blockSize = 4
const numStreams = 5

type streamClient struct {
	client mixmessages.Node_StreamPostPhaseClient
	cancel context.CancelFunc
}

// StreamTransmitPhase streams slot messages to the provided Node.
func StreamTransmitPhase(roundID id.Round, serverInstance phase.GenericInstance, getChunk phase.GetChunk,
	getMessage phase.GetMessage) error {

	instance, ok := serverInstance.(*internal.Instance)
	if !ok {
		return errors.Errorf("Invalid server instance passed in")
	}
	//get the round so you can get its batch size
	r, err := instance.GetRoundManager().GetRound(roundID)
	if err != nil {
		return errors.Errorf("Could not retrieve round %d from"+
			" manager %s", roundID, err)
	}
	rType := r.GetCurrentPhaseType()
	topology := r.GetTopology()
	nodeID := instance.GetID()

	// Pull the particular server host object from the commManager
	recipientID := topology.GetNextNode(nodeID)
	recipientIndex := topology.GetNodeLocation(recipientID)
	recipient := topology.GetHostAtIndex(recipientIndex)
	header := mixmessages.BatchInfo{
		Round: &mixmessages.RoundInfo{
			ID: uint64(roundID),
		},
		FromPhase: int32(r.GetCurrentPhaseType()),
		BatchSize: r.GetBatchSize(),
	}

	// get the current phase
	// get this here to use down below to record the measurment to stop a race
	// conditions where other nodes finish their works and get this node to
	// iterate phase before the measure code runs
	currentPhase := r.GetCurrentPhase()

	streamClients := make([]streamClient, numStreams)
	wg := sync.WaitGroup{}

	for i := 0; i < numStreams; i++ {
		wg.Add(1)
		localI := i
		go func() {
			sc, c, err := instance.GetNetwork().GetPostPhaseStreamClient(
				recipient, header)
			if err != nil {
				jww.FATAL.Panicf("Error on comm, unable to get streaming "+
					"client: %+v", err)
			}
			streamClients[localI] = streamClient{
				sc,
				c,
			}
			wg.Done()
		}()
	}

	//pull the first chunk reception out so it can be timestmaped
	chunk, finish := getChunk()
	start := time.Now()
	// For each message chunk (slot) stream it out
	sendBuff := make([]*mixmessages.Slot, 0, blockSize)

	packetSizeSum := 0

	currentSC := 0

	for ; finish; chunk, finish = getChunk() {
		for i := chunk.Begin(); i < chunk.End(); i++ {
			msg := getMessage(i)
			marshaled, _ := proto.Marshal(msg)
			packetSizeSum += len(marshaled)
			sendBuff = append(sendBuff, msg)

			if len(sendBuff) == blockSize {
				err = streamClients[currentSC].client.Send(&mixmessages.Slots{Messages: sendBuff})
				if err != nil {
					return errors.Errorf("Error on comm, not able to send "+
						"slot: %+v", err)
				}
				sendBuff = make([]*mixmessages.Slot, 0, blockSize)
				currentSC++
				if currentSC == numStreams {
					currentSC = 0
				}
			}
		}
	}

	if len(sendBuff) > 0 {
		err = streamClients[currentSC].client.Send(&mixmessages.Slots{Messages: sendBuff})
		if err != nil {
			return errors.Errorf("Error on comm, not able to send "+
				"slot: %+v", err)
		}
	}

	wg.Wait()

	end := time.Now()

	measureFunc := currentPhase.Measure
	if measureFunc != nil {
		measureFunc(measure.TagTransmitLastSlot)
	}

	wg = sync.WaitGroup{}
	// Receive ack and cancel client streaming context
	for i := 0; i < numStreams; i++ {
		localI := i
		wg.Add(1)
		go func() {
			ack, err := streamClients[localI].client.CloseAndRecv()
			if err != nil {
				if err != nil {
					jww.FATAL.Panicf("Received error from closing stream %d: %s", localI, err)
				}

				// Make sure the comm doesn't return an Ack with an error message
				if ack != nil && ack.Error != "" {
					jww.FATAL.Panicf("Received ack error from closing stream %d: %s", localI, ack.Error)
				}
			}
			streamClients[localI].cancel()
			wg.Done()
		}()
	}

	wg.Wait()

	localServer := instance.GetNetwork().String()
	port := strings.Split(localServer, ":")[1]
	addr := fmt.Sprintf("%s:%s", nodeID, port)
	name := services.NameStringer(addr, topology.GetNodeLocation(nodeID),
		topology.Len())

	jww.INFO.Printf("[%s] RID %d StreamTransmitPhase FOR \"%s\""+
		" COMPLETE/SEND", name, roundID, rType)

	jww.INFO.Printf("\tbwLogging: Round %d, "+
		"transmitted phase: %s, "+
		"from: %s, to: %s, "+
		"started: %v, "+
		"ended: %v, "+
		"duration: %d, packetSize: %d",
		roundID, currentPhase.GetType(),
		instance.GetID(), recipientID,
		start, end, end.Sub(start).Milliseconds(),
		packetSizeSum/int(r.GetBatchSize()))

	return nil
}

// StreamPostPhase implements the server gRPC handler for posting a
// phase from another node
func StreamPostPhase(ch chan []*mixmessages.Slot, batchSize uint32,
	stream mixmessages.Node_StreamPostPhaseServer) (time.Time, time.Time, error) {
	// Send a chunk for each slot received along with
	// its index until an error is received
	slots, err := stream.Recv()
	slotsReceived := uint32(0)
	var start, end time.Time

	last := time.Now()

	for ; err == nil; slots, err = stream.Recv() {
		now := time.Now()
		jww.INFO.Printf("Reception Delta: %d ns", now.Sub(last).Nanoseconds())
		last = now
		for i := range slots.Messages {
			ch <- []*mixmessages.Slot{slots.Messages[i]}
			slotsReceived++
			if slotsReceived >= batchSize && end.Equal(time.Time{}) {
				end = time.Now()
			} else if slotsReceived == 1 {
				start = time.Now()
			}
		}
	}

	// Set error in ack message if we didn't receive all slots
	ack := messages.Ack{
		Error: "",
	}
	if err != io.EOF {
		ack.Error = fmt.Sprintf("errors occurred, %v/%v slots "+
			"recived: %s", slotsReceived, batchSize, err.Error())
	} /*else if slotsReceived != batchSize {
		ack.Error = fmt.Sprintf("Mismatch between batch size %v"+
			"and received num slots %v, no error", slotsReceived, batchSize)
	}*/

	// Close the stream by sending ack
	// and returning whether it succeeded
	errClose := stream.SendAndClose(&ack)

	if errClose != nil && ack.Error != "" {
		return start, end, errors.WithMessage(errClose, ack.Error)
	} else if errClose == nil && ack.Error != "" {
		return start, end, errors.New(ack.Error)
	} else {
		return start, end, errClose
	}
}
