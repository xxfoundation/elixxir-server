////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package io

// FIXME: this import list makes it feel like the api is spaghetti
import (
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/server/phase"
	"gitlab.com/elixxir/server/services"
	"sync"
	"testing"
)

//Test that post phase properly sends the results to the phase via mockPhase
func TestPostPhase(t *testing.T) {

	numSlots := 3

	//Get a mock phase
	p := &MockPhase{}

	//Build a mock batch to receive
	batch := mixmessages.Batch{}

	for i := 0; i < numSlots; i++ {
		batch.Slots = append(batch.Slots,
			&mixmessages.Slot{
				MessagePayload: []byte{byte(i)},
			})
	}

	//receive the batch
	err := PostPhase(p, &batch)

	if err != nil {
		t.Errorf("PostPhase: Unexpected error returned: %+v", err)
	}

	for index := range batch.Slots {
		if p.chunks[index].Begin() != uint32(index) {
			t.Errorf("PostPhase: output chunk not equal to passed;"+
				"Expected: %v, Recieved: %v", index, p.chunks[index].Begin())
		}

		if p.indices[index] != uint32(index) {
			t.Errorf("PostPhase: output index  not equal to passed;"+
				"Expected: %v, Recieved: %v", index, p.indices[index])
		}
	}

	batch.Slots[0].Salt = []byte{42}

	err = PostPhase(p, &batch)

	if err == nil {
		t.Errorf("PostPhase: did not error when expected")
	}
}

var receivedBatch *mixmessages.Batch
var done sync.Mutex

//Tests that a batch sent via transmit phase arrives correctly
func TestTransmitPhase(t *testing.T) {

	//Setup the network
	comms, topology := buildTestNetworkComponents(
		[]func() *node.Implementation{nil, MockCommImplementation})
	defer Shutdown(comms)

	//Build the mock functions called by the transimitter
	chunkCnt := uint32(0)
	batchSize := uint32(5)
	roundID := id.Round(5)
	phaseTy := phase.Type(2)

	getChunk := func() (services.Chunk, bool) {
		if chunkCnt < batchSize {
			chunk, ok := services.NewChunk(chunkCnt, chunkCnt+1), false
			chunkCnt++
			return chunk, ok
		}
		return services.NewChunk(0, 0), true
	}

	getMsg := func(index uint32) *mixmessages.Slot {
		return &mixmessages.Slot{MessagePayload: []byte{0}}
	}

	done.Lock()

	//call the transmitter
	err := TransmitPhase(comms[0], batchSize, roundID, phaseTy, getChunk,
		getMsg, topology, topology.GetNodeAtIndex(0))

	if err != nil {
		t.Errorf("TransmitPhase: Unexpected error: %+v", err)
	}

	//Use lock to wait until handler receives results
	done.Lock()
	defer done.Unlock()

	//Check that what was received is correct
	if id.Round(receivedBatch.Round.ID) != roundID {
		t.Errorf("TransmitPhase: Incorrect round ID"+
			"Expected: %v, Recieved: %v", roundID, receivedBatch.Round.ID)
	}

	if phase.Type(receivedBatch.ForPhase) != phaseTy {
		t.Errorf("TransmitPhase: Incorrect Phase type"+
			"Expected: %v, Recieved: %v", phaseTy, receivedBatch.ForPhase)
	}

	if uint32(len(receivedBatch.Slots)) != batchSize {
		t.Errorf("TransmitPhase: Recieved Batch of wrong size"+
			"Expected: %v, Recieved: %v", batchSize,
			uint32(len(receivedBatch.Slots)))
	}
}

func MockCommImplementation() *node.Implementation {
	impl := node.NewImplementation()
	impl.Functions.PostPhase = func(batch *mixmessages.Batch) {
		receivedBatch = batch
		done.Unlock()
	}
	return impl
}
