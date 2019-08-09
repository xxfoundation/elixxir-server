package io

import (
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/large"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/server/phase"
	"gitlab.com/elixxir/server/server/round"
	"gitlab.com/elixxir/server/services"
	"reflect"
	"testing"
	"time"
)

func TestPostPrecompResult_Errors(t *testing.T) {
	grp := cyclic.NewGroup(large.NewIntFromString(primeString, 16),
		large.NewInt(2), large.NewInt(97))
	r := round.NewBuffer(grp, 5, 5)

	// If the number of slots doesn't match the batch, there should be an error
	err := PostPrecompResult(r, grp, []*mixmessages.Slot{})
	if err == nil {
		t.Error("No error from batch size mismatch")
	}
}

func TestPostPrecompResult(t *testing.T) {
	// This test actually overwrites the precomputations for a round
	grp := cyclic.NewGroup(large.NewIntFromString(primeString, 16),
		large.NewInt(2), large.NewInt(97))
	const bs = 5
	r := round.NewBuffer(grp, bs, bs)

	// There should be no error in this case, because there are enough slots
	var slots []*mixmessages.Slot
	const start = 2
	for precompValue := start; precompValue < bs+start; precompValue++ {
		slots = append(slots, &mixmessages.Slot{
			EncryptedPayloadAKeys: grp.NewInt(int64(precompValue)).
				Bytes(),
			EncryptedPayloadBKeys: grp.NewInt(int64(precompValue + bs)).
				Bytes(),
		})
	}

	err := PostPrecompResult(r, grp, slots)
	if err != nil {
		t.Error(err)
	}

	// Then, the slots in the round buffer should be set to those integers
	for precompValue := start; precompValue < bs+start; precompValue++ {
		index := uint32(precompValue - start)
		payloadAPrecomp := r.PayloadAPrecomputation.Get(index)
		if payloadAPrecomp.Cmp(grp.NewInt(int64(precompValue))) != 0 {
			t.Errorf("payload A precomp didn't match at index %v;"+
				"Expected: %v, Recieved: %v", index, precompValue,
				payloadAPrecomp.Text(16))
		}
		payloadBPrecomp := r.PayloadBPrecomputation.Get(index)
		if payloadBPrecomp.Cmp(grp.NewInt(int64(precompValue+bs))) != 0 {
			t.Errorf("payload B precomp didn't match at index %v;"+
				"Expected: %v, Recieved: %v", index, precompValue+bs,
				payloadBPrecomp.Text(16))
		}
	}
}

func MockPostPrecompResultImplementation(
	precompReceiver chan []*mixmessages.Slot,
	roundReceiver chan uint64) *node.Implementation {
	impl := node.NewImplementation()
	impl.Functions.PostPrecompResult = func(roundID uint64, slots []*mixmessages.Slot) error {
		roundReceiver <- roundID
		precompReceiver <- slots
		return nil
	}

	return impl
}

func getMockPostPrecompSlot(i uint32) *mixmessages.Slot {
	return &mixmessages.Slot{
		PartialPayloadACypherText: []byte{byte(i)},
		PartialPayloadBCypherText: []byte{byte(i)},
	}
}

// Tests happy path of the PostPrecompResult transmission handler
func TestTransmitPostPrecompResult(t *testing.T) {
	//Setup the network
	const numNodes = 5
	numReceivedRounds := 0
	numReceivedPrecomps := 0
	roundReceiver := make(chan uint64, numNodes)
	precompReceiver := make(chan []*mixmessages.Slot, numNodes)
	comms, topology := buildTestNetworkComponents(
		[]*node.Implementation{
			// All five nodes should receive the same message
			MockPostPrecompResultImplementation(precompReceiver, roundReceiver),
			MockPostPrecompResultImplementation(precompReceiver, roundReceiver),
			MockPostPrecompResultImplementation(precompReceiver, roundReceiver),
			MockPostPrecompResultImplementation(precompReceiver, roundReceiver),
			MockPostPrecompResultImplementation(precompReceiver, roundReceiver)}, 50)
	defer Shutdown(comms)

	rndID := id.Round(42)
	batchSize := uint32(5)

	slotCount := uint32(0)

	getchunk := func() (services.Chunk, bool) {

		chunk := services.NewChunk(slotCount, slotCount+1)

		good := true

		if slotCount >= batchSize {
			good = false
		}

		slotCount++

		return chunk, good
	}

	err := TransmitPrecompResult(comms[numNodes-1], batchSize,
		rndID, phase.PrecompReveal, getchunk, getMockPostPrecompSlot,
		topology, nil, nil)

	if err != nil {
		t.Errorf("TransmitPrecompResult: Unexpected error: %+v", err)
	}

	// Make sure that everything that was supposed to come through does come
	// through
Loop:
	for {
		select {
		// TODO also receive from the precomp receiver
		case receivedRoundID := <-roundReceiver:
			if receivedRoundID != uint64(rndID) {
				t.Errorf("TransmitPrecompResult: Incorrect round ID"+
					"Expected: %v, Received: %v", rndID, receivedRoundID)
			}
			numReceivedRounds++
		case receivedPrecomp := <-precompReceiver:
			// Construct expected mock precomp result
			expectedPrecompResults := make([]*mixmessages.Slot, numNodes)
			for i := uint32(0); i < numNodes; i++ {
				expectedPrecompResults[i] = getMockPostPrecompSlot(i)
			}
			if !reflect.DeepEqual(receivedPrecomp, expectedPrecompResults) {
				t.Error("Precomps differed")
			}
			numReceivedPrecomps++
		case <-time.After(5 * time.Second):
			t.Errorf("Test timed out!")
			break Loop
		}
		if numReceivedRounds >= numNodes && numReceivedPrecomps >= numNodes {
			break Loop
		}
	}
}
