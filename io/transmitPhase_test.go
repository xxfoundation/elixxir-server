///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package io

import (
	"fmt"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/comms/testkeys"
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/server/globals"
	"gitlab.com/elixxir/server/internal"
	"gitlab.com/elixxir/server/internal/measure"
	"gitlab.com/elixxir/server/internal/phase"
	"gitlab.com/elixxir/server/internal/round"
	"gitlab.com/elixxir/server/internal/state"
	"gitlab.com/elixxir/server/services"
	"gitlab.com/elixxir/server/testUtil"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/utils"
	"math/rand"
	"testing"
)

// Test that post phase properly sends the results to the phase via mockPhase
func TestPostPhase(t *testing.T) {

	numSlots := 3

	//Get a mock phase
	mockPhase := &MockPhase{}

	//Build a mock mockBatch to receive
	mockBatch := mixmessages.Batch{}

	for i := 0; i < numSlots; i++ {
		mockBatch.Slots = append(mockBatch.Slots,
			&mixmessages.Slot{
				PayloadA: []byte{byte(i)},
			})
	}

	//receive the mockBatch
	err := PostPhase(mockPhase, &mockBatch)

	if err != nil {
		t.Errorf("PostPhase: Unexpected error returned: %+v", err)
	}

	for index := range mockBatch.Slots {
		if mockPhase.chunks[index].Begin() != uint32(index) {
			t.Errorf("PostPhase: output chunk not equal to passed;"+
				"Expected: %v, Received: %v", index, mockPhase.chunks[index].Begin())
		}

		if mockPhase.indices[index] != uint32(index) {
			t.Errorf("PostPhase: output index  not equal to passed;"+
				"Expected: %v, Received: %v", index, mockPhase.indices[index])
		}
	}

	mockBatch.Slots[0].Salt = []byte{42}
	mockBatch.Round = &mixmessages.RoundInfo{}

	err = PostPhase(mockPhase, &mockBatch)

	if err == nil {
		t.Errorf("PostPhase: did not error when expected")
	}
}

var receivedBatch *mixmessages.Batch

// Tests that a batch sent via transmit phase arrives correctly
func TestTransmitPhase(t *testing.T) {
	instance, nodeAddr := mockInstance(t, mockPostPhaseImplementation)

	// Build the mock functions called by the transmitter
	chunkCnt := uint32(0)
	batchSize := uint32(5)
	roundID := id.Round(5)
	phaseTy := phase.Type(0)

	response := phase.NewResponse(phase.ResponseDefinition{
		PhaseAtSource:  phase.PrecompGeneration,
		ExpectedStates: []phase.State{phase.Active},
		PhaseToExecute: phase.PrecompGeneration})

	grp := initImplGroup()
	p := testUtil.InitMockPhase(t)
	p.Ptype = phase.PrecompGeneration
	responseMap := make(phase.ResponseMap)
	responseMap["RealPermuteVerification"] = response

	topology := connect.NewCircuit([]*id.ID{instance.GetID()})

	cert, _ := utils.ReadFile(testkeys.GetNodeCertPath())
	nodeHost, _ := connect.NewHost(instance.GetID(), nodeAddr, cert, connect.GetDefaultHostParams())
	topology.AddHost(nodeHost)
	_, err := instance.GetNetwork().AddHost(instance.GetID(), nodeAddr, cert, connect.GetDefaultHostParams())
	if err != nil {
		t.Errorf("Failed to add host to instance: %v", err)
	}

	rnd, err := round.New(grp, nil, roundID, []phase.Phase{p}, responseMap, topology,
		topology.GetNodeAtIndex(0), batchSize, instance.GetRngStreamGen(), nil,
		"0.0.0.0", nil, nil)
	if err != nil {
		t.Error()
	}

	instance.GetRoundManager().AddRound(rnd)

	getChunk := func() (services.Chunk, bool) {
		if chunkCnt < batchSize {
			chunk := services.NewChunk(chunkCnt, chunkCnt+1)
			chunkCnt++
			return chunk, true
		}
		return services.NewChunk(0, 0), false
	}
	getMsg := func(index uint32) *mixmessages.Slot {
		return &mixmessages.Slot{PayloadA: []byte{0}}
	}

	//call the transmitter
	err = TransmitPhase(roundID, instance, getChunk, getMsg)

	if err != nil {
		t.Errorf("TransmitPhase: Unexpected error: %+v", err)
	}

	//Check that what was receivedFinishRealtime is correct
	if id.Round(receivedBatch.Round.ID) != roundID {
		t.Errorf("TransmitPhase: Incorrect round ID"+
			"Expected: %v, Received: %v", roundID, receivedBatch.Round.ID)
	}

	if phase.Type(receivedBatch.FromPhase) != phaseTy {
		t.Errorf("TransmitPhase: Incorrect Phase type"+
			"Expected: %v, Received: %v", phaseTy, receivedBatch.FromPhase)
	}

	if uint32(len(receivedBatch.Slots)) != batchSize {
		t.Errorf("TransmitPhase: Received Batch of wrong size"+
			"Expected: %v, Received: %v", batchSize,
			uint32(len(receivedBatch.Slots)))
	}
}

func mockPostPhaseImplementation(instance *internal.Instance) *node.Implementation {
	impl := node.NewImplementation()
	impl.Functions.PostPhase = func(batch *mixmessages.Batch, auth *connect.Auth) error {
		receivedBatch = batch
		return nil
	}
	return impl
}

var cnt = 0

func mockInstance(t interface{}, impl func(instance *internal.Instance) *node.Implementation) (*internal.Instance, string) {
	switch v := t.(type) {
	case *testing.T:
	case *testing.M:
		break
	default:
		panic(fmt.Sprintf("Cannot use outside of test environment; %+v", v))
	}

	cert, _ := utils.ReadFile(testkeys.GetNodeCertPath())
	key, _ := utils.ReadFile(testkeys.GetNodeKeyPath())

	nid := internal.GenerateId(t)

	var err error

	//make registration rsa key pair
	regPKey, err := rsa.GenerateKey(csprng.NewSystemRNG(), 1024)
	if err != nil {
		panic(fmt.Sprintf("Could not generate registration private key: %+v", err))
	}

	//make server rsa key pair
	pk, _ := utils.ReadFile(testkeys.GetNodeKeyPath())

	privKey, _ := rsa.LoadPrivateKeyFromPem(pk)

	//serverRSAPub := serverRSAPriv.GetPublic()
	nodeAddr := fmt.Sprintf("0.0.0.0:%d", 7000+rand.Intn(1000)+cnt)

	cnt++

	def := internal.Definition{
		ID:               nid,
		UserRegistry:     &globals.UserMap{},
		ResourceMonitor:  &measure.ResourceMonitor{},
		PrivateKey:       privKey,
		PublicKey:        privKey.GetPublic(),
		TlsCert:          cert,
		TlsKey:           key,
		FullNDF:          testUtil.NDF,
		PartialNDF:       testUtil.NDF,
		ListeningAddress: nodeAddr,
	}

	def.Permissioning.PublicKey = regPKey.GetPublic()
	nodeIDs := make([]*id.ID, 0)
	nodeIDs = append(nodeIDs, nid)
	def.Gateway.ID = &id.TempGateway
	def.Gateway.ID.SetType(id.Gateway)

	mach := state.NewTestMachine(dummyStates, current.PRECOMPUTING, t)

	instance, _ := internal.CreateServerInstance(&def, impl, mach, "1.1.0")

	return instance, nodeAddr

}
