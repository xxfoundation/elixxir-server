////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package io

import (
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/comms/testkeys"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/server/internal"
	"gitlab.com/elixxir/server/internal/phase"
	"gitlab.com/elixxir/server/internal/round"
	"gitlab.com/elixxir/server/services"
	"gitlab.com/elixxir/server/testUtil"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/crypto/large"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/utils"
	"testing"
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
				"Expected: %v, Received: %v", index, precompValue,
				payloadAPrecomp.Text(16))
		}
		payloadBPrecomp := r.PayloadBPrecomputation.Get(index)
		if payloadBPrecomp.Cmp(grp.NewInt(int64(precompValue+bs))) != 0 {
			t.Errorf("payload B precomp didn't match at index %v;"+
				"Expected: %v, Received: %v", index, precompValue+bs,
				payloadBPrecomp.Text(16))
		}
	}
}

func MockPostPrecompResultImplementation(instance *internal.Instance) *node.Implementation {
	impl := node.NewImplementation()
	impl.Functions.PostPrecompResult = func(roundID uint64, numslots uint32, auth *connect.Auth) error {
		roundReceiver <- roundID
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

var roundReceiver chan uint64

// Tests happy path of the PostPrecompResult transmission handler
func TestTransmitPostPrecompResult(t *testing.T) {
	//fixme: modify test to check multiple nodes
	// need to set up multiple instances, set topology, and add hosts to topology and to instances

	//Setup the network
	const numNodes = 1
	var instances []*internal.Instance
	var nodeAddr string
	var instance *internal.Instance
	for i := 0; i < numNodes; i++ {
		instance, nodeAddr = mockInstance(t, MockPostPrecompResultImplementation)
		instances = append(instances, instance)
	}

	instance = instances[0]

	roundReceiver = make(chan uint64, numNodes)

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

	topology := connect.NewCircuit([]*id.ID{instance.GetID()})

	cert, _ := utils.ReadFile(testkeys.GetNodeCertPath())
	nodeHost, _ := connect.NewHost(instance.GetID(), nodeAddr, cert, connect.GetDefaultHostParams())
	topology.AddHost(nodeHost)
	_, err := instance.GetNetwork().AddHost(instance.GetID(), nodeAddr, cert, connect.GetDefaultHostParams())
	if err != nil {
		t.Errorf("Failed to add host to instance: %v", err)
	}

	rnd, err := round.New(grp, rndID, []phase.Phase{p}, responseMap, topology, topology.GetNodeAtIndex(0), batchSize, instance.GetRngStreamGen(), nil, "0.0.0.0", nil, nil, nil, nil)
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
}
