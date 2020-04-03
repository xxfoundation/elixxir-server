package io

import (
	"gitlab.com/elixxir/comms/connect"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/comms/testkeys"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/large"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/primitives/utils"
	"gitlab.com/elixxir/server/server"
	"gitlab.com/elixxir/server/server/phase"
	"gitlab.com/elixxir/server/server/round"
	"gitlab.com/elixxir/server/services"
	"gitlab.com/elixxir/server/testUtil"
	"reflect"
	"testing"
	"time"
)

func TestPostPrecompResult_Errors(t *testing.T) {
	grp := cyclic.NewGroup(large.NewIntFromString(primeString, 16),
		large.NewInt(2))
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
		large.NewInt(2))
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

func MockPostPrecompResultImplementation(instance *server.Instance) *node.Implementation {
	impl := node.NewImplementation()
	impl.Functions.PostPrecompResult = func(roundID uint64, slots []*mixmessages.Slot, auth *connect.Auth) error {
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

func postPrecompInstance() {

}

var roundReceiver chan uint64
var precompReceiver chan []*mixmessages.Slot

// Tests happy path of the PostPrecompResult transmission handler
func TestTransmitPostPrecompResult(t *testing.T) {
	//fixme: modify test to check multiple nodes
	// need to set up multiple instances, set topology, and add hosts to topology and to instances

	//Setup the network
	const numNodes = 1
	var instances []*server.Instance
	var nodeAddr string
	var instance *server.Instance
	for i := 0; i < numNodes; i++ {
		instance, nodeAddr = mockInstance(t, MockPostPrecompResultImplementation)
		instances = append(instances, instance)
	}

	instance = instances[0]

	numReceivedRounds := 0
	numReceivedPrecomps := 0
	roundReceiver = make(chan uint64, numNodes)
	precompReceiver = make(chan []*mixmessages.Slot, numNodes)

	rndID := id.Round(1)
	batchSize := uint32(1)

	slotCount := uint32(0)

	response := phase.NewResponse(phase.ResponseDefinition{
		PhaseAtSource:  phase.PrecompDecrypt,
		ExpectedStates: []phase.State{phase.Active},
		PhaseToExecute: phase.PrecompDecrypt})

	grp := initImplGroup()

	p := testUtil.InitMockPhase(t)
	p.Ptype = phase.PrecompDecrypt
	responseMap := make(phase.ResponseMap)
	responseMap["PrecompDecrypt"] = response

	topology := connect.NewCircuit([]*id.Node{instance.GetID()})

	cert, _ := utils.ReadFile(testkeys.GetNodeCertPath())
	nodeHost, _ := connect.NewHost(instance.GetID().String(), nodeAddr, cert, false, true)
	topology.AddHost(nodeHost)
	_, err := instance.GetNetwork().AddHost(instance.GetID().String(), nodeAddr, cert, false, true)
	if err != nil {
		t.Errorf("Failed to add host to instance: %v", err)
	}

	rnd, err := round.New(grp, nil, rndID, []phase.Phase{p}, responseMap, topology,
		topology.GetNodeAtIndex(0), batchSize, instance.GetRngStreamGen(),
		"0.0.0.0")
	if err != nil {
		t.Errorf("Failed to create round: %v", err)
	}

	instance.GetRoundManager().AddRound(rnd)

	getchunk := func() (services.Chunk, bool) {

		chunk := services.NewChunk(slotCount, slotCount+1)

		good := true

		if slotCount >= batchSize {
			good = false
		}

		slotCount++

		return chunk, good
	}

	err = TransmitPrecompResult(rndID, instance, getchunk, getMockPostPrecompSlot)

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
				t.Errorf("Precomps differed: Expected: %v\n\tRecieved: %v", expectedPrecompResults, receivedPrecomp)
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
