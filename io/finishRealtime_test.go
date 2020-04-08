////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package io

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/comms/connect"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/large"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/server/phase"
	"gitlab.com/elixxir/server/server/round"
	"gitlab.com/elixxir/server/services"
	"gitlab.com/elixxir/server/testUtil"
	"testing"
	"time"
)

var receivedFinishRealtime = make(chan *mixmessages.RoundInfo, 100)
var getMessage = func(index uint32) *mixmessages.Slot {
	return &mixmessages.Slot{}
}

func MockFinishRealtimeImplementation() *node.Implementation {
	impl := node.NewImplementation()
	impl.Functions.FinishRealtime = func(message *mixmessages.RoundInfo, auth *connect.Auth) error {
		receivedFinishRealtime <- message
		return nil
	}
	return impl
}

// Test that TransmitFinishRealtime correctly broadcasts message
// to all other nodes
func TestSendFinishRealtime(t *testing.T) {
	instance, _, _, _, _, _, _ := setup(t)

	//Setup the network
	numNodes := 4
	numRecv := 0
	comms, topology := buildTestNetworkComponents(
		[]*node.Implementation{
			MockFinishRealtimeImplementation(),
			MockFinishRealtimeImplementation(),
			MockFinishRealtimeImplementation(),
			MockFinishRealtimeImplementation(),
			MockFinishRealtimeImplementation()}, 0)
	defer Shutdown(comms)

	const numSlots = 10

	const numChunks = numSlots / 2

	response := phase.NewResponse(phase.ResponseDefinition{
		PhaseAtSource:  phase.RealPermute,
		ExpectedStates: []phase.State{phase.Active},
		PhaseToExecute: phase.RealPermute})

	roundID := id.Round(0)
	grp := initImplGroup()
	p := testUtil.InitMockPhase(t)
	p.Ptype = phase.RealPermute
	responseMap := make(phase.ResponseMap)
	responseMap["RealPermuteVerification"] = response

	rnd, err := round.New(grp, nil, roundID, []phase.Phase{p}, responseMap, topology,
		topology.GetNodeAtIndex(0), numSlots, instance.GetRngStreamGen(),
		"0.0.0.0")
	if err != nil {
		t.Error()
	}
	instance.GetRoundManager().AddRound(rnd)

	chunkInputChan := make(chan services.Chunk, numChunks)

	getChunk := func() (services.Chunk, bool) {
		chunk, ok := <-chunkInputChan
		return chunk, ok
	}
	errCH := make(chan error)

	go func() {
		err = TransmitFinishRealtime(roundID, instance, getChunk, getMessage)
		errCH <- err
	}()

	for i := 0; i < numChunks; i++ {
		chunkInputChan <- services.NewChunk(uint32(i*2), uint32(i*2+1))
	}

	close(chunkInputChan)

	var cr *round.CompletedRound

	for cr == nil {
		cr, _ = instance.GetCompletedBatchQueue().Receive()
		time.Sleep(1 * time.Millisecond)
	}

	if len(cr.Round) != numSlots {
		t.Errorf("TransmitFinishRealtime: did not recieve the correct: "+
			"number of chunks; expected: %v, recieved: %v", numSlots,
			len(cr.Round))
	}

	goErr := <-errCH

	if goErr != nil {
		t.Errorf("TransmitFinishRealtime: Unexpected error: %+v", err)
	}

Loop:
	for {
		select {
		case msg := <-receivedFinishRealtime:
			if id.Round(msg.ID) != roundID {
				t.Errorf("TransmitFinishRealtime: Incorrect round ID"+
					"Expected: %v, Received: %v", roundID, msg.ID)
			}
			numRecv++
			if numRecv == numNodes {
				break Loop
			}
		case <-time.After(5 * time.Second):
			t.Errorf("Test timed out!")
			break Loop
		}
	}
}

func MockFinishRealtimeImplementation_Error() *node.Implementation {
	impl := node.NewImplementation()
	impl.Functions.FinishRealtime = func(message *mixmessages.RoundInfo, auth *connect.Auth) error {
		return errors.New("Test error")
	}
	return impl
}

func TestTransmitFinishRealtime_Error(t *testing.T) {
	instance, _, _, _, _, _, _ := setup(t)

	//Setup the network
	comms, topology := buildTestNetworkComponents(
		[]*node.Implementation{
			MockFinishRealtimeImplementation_Error(),
			MockFinishRealtimeImplementation_Error(),
			MockFinishRealtimeImplementation_Error(),
			MockFinishRealtimeImplementation_Error(),
			MockFinishRealtimeImplementation_Error()}, 0)
	defer Shutdown(comms)

	const numSlots = 10
	const numChunks = numSlots / 2

	response := phase.NewResponse(phase.ResponseDefinition{
		PhaseAtSource:  phase.RealPermute,
		ExpectedStates: []phase.State{phase.Active},
		PhaseToExecute: phase.RealPermute})

	roundID := id.Round(0)
	grp := initImplGroup()
	p := testUtil.InitMockPhase(t)
	p.Ptype = phase.RealPermute
	responseMap := make(phase.ResponseMap)
	responseMap["RealPermuteVerification"] = response

	rnd, err := round.New(grp, nil, roundID, []phase.Phase{p}, responseMap, topology,
		topology.GetNodeAtIndex(0), numSlots, instance.GetRngStreamGen(),
		"0.0.0.0")
	if err != nil {
		t.Error()
	}

	instance.GetRoundManager().AddRound(rnd)

	chunkInputChan := make(chan services.Chunk, numChunks)

	getChunk := func() (services.Chunk, bool) {
		chunk, ok := <-chunkInputChan
		return chunk, ok
	}
	errCH := make(chan error)

	go func() {
		err := TransmitFinishRealtime(roundID, instance, getChunk, getMessage)
		errCH <- err
	}()

	go func() {
		for i := 0; i < numChunks; i++ {
			chunkInputChan <- services.NewChunk(uint32(i*2), uint32(i*2+1))
		}

		close(chunkInputChan)
	}()

	var cr *round.CompletedRound

	for cr == nil {
		cr, _ = instance.GetCompletedBatchQueue().Receive()
		time.Sleep(1 * time.Millisecond)
	}

	if len(cr.Round) != numSlots {
		t.Errorf("TransmitFinishRealtime: did not recieve the correct: "+
			"number of chunks; expected: %v, recieved: %v", numSlots,
			len(cr.Round))
	}

	goErr := <-errCH

	if goErr == nil {
		t.Error("SendFinishRealtime: error did not occur when provoked")
	}
}

func initImplGroup() *cyclic.Group {
	primeString := "FFFFFFFFFFFFFFFFC90FDAA22168C234C4C6628B80DC1CD1" +
		"29024E088A67CC74020BBEA63B139B22514A08798E3404DD" +
		"EF9519B3CD3A431B302B0A6DF25F14374FE1356D6D51C245" +
		"E485B576625E7EC6F44C42E9A637ED6B0BFF5CB6F406B7ED" +
		"EE386BFB5A899FA5AE9F24117C4B1FE649286651ECE45B3D" +
		"C2007CB8A163BF0598DA48361C55D39A69163FA8FD24CF5F" +
		"83655D23DCA3AD961C62F356208552BB9ED529077096966D" +
		"670C354E4ABC9804F1746C08CA18217C32905E462E36CE3B" +
		"E39E772C180E86039B2783A2EC07A28FB5C55DF06F4C52C9" +
		"DE2BCBF6955817183995497CEA956AE515D2261898FA0510" +
		"15728E5A8AACAA68FFFFFFFFFFFFFFFF"
	grp := cyclic.NewGroup(large.NewIntFromString(primeString, 16), large.NewInt(2))
	return grp
}
