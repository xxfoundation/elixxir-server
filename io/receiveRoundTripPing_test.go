////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package io

import (
	"crypto/rand"
	"github.com/golang/protobuf/ptypes"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/globals"
	"gitlab.com/elixxir/server/internal"
	"gitlab.com/elixxir/server/internal/measure"
	"gitlab.com/elixxir/server/internal/phase"
	"gitlab.com/elixxir/server/internal/round"
	"gitlab.com/elixxir/server/testUtil"
	"testing"
)

func TestReceiveRoundTripPing(t *testing.T) {
	grp := initImplGroup()
	const numNodes = 5

	resourceMonitor := measure.ResourceMonitor{}
	resourceMonitor.Set(measure.ResourceMetric{})
	//Dummy round object
	newRound := round.NewDummyRound(id.Round(1), 10, t)

	// Set instance for first node
	def := internal.Definition{
		UserRegistry:    &globals.UserMap{},
		ResourceMonitor: &resourceMonitor,
	}
	def.ID = newRound.GetTopology().GetNodeAtIndex(0)

	instance, _ := mockServerInstance(t, current.PRECOMPUTING)

	// Set up a round first node
	roundID := id.Round(45)

	mockPhase := testUtil.InitMockPhase(t)
	mockPhase.Ptype = phase.PrecompShare

	tagKey := mockPhase.GetType().String() + "Verification"
	responseMap := make(phase.ResponseMap)
	responseMap[tagKey] = phase.NewResponse(
		phase.ResponseDefinition{
			PhaseAtSource:  mockPhase.GetType(),
			ExpectedStates: []phase.State{phase.Active},
			PhaseToExecute: mockPhase.GetType()},
	)

	batchSize := uint32(11)

	r, err := round.New(grp, &globals.UserMap{}, roundID, []phase.Phase{mockPhase},
		responseMap, newRound.GetTopology(), newRound.GetTopology().GetNodeAtIndex(0), batchSize,
		instance.GetRngStreamGen(), nil, "0.0.0.0", nil)
	if err != nil {
		t.Errorf("Failed to create new round: %+v", err)
		return
	}
	r.StartRoundTrip("test")

	before := r.GetRTEnd().String()

	instance.GetRoundManager().AddRound(r)
	A := make([]byte, 150)
	B := make([]byte, 150)
	_, err = rand.Read(A)
	if err != nil {
		t.Errorf("TransmitRoundTripPing: failed to generate random bytes A: %+v", err)
	}
	_, err = rand.Read(B)
	if err != nil {
		t.Errorf("TransmitRoundTripPing: failed to generate random bytes B: %+v", err)
	}
	anyPayload := &pb.Batch{
		Slots: []*pb.Slot{
			{
				SenderID: instance.GetID().Bytes(),
				PayloadA: A,
				PayloadB: B,
				// Because the salt is just one byte,
				// this should fail in the Realtime Decrypt graph.
				Salt:  make([]byte, 32),
				KMACs: [][]byte{{5}},
			},
		},
	}
	any, _ := ptypes.MarshalAny(anyPayload)

	msg := &pb.RoundTripPing{
		Payload: any,
		Round: &pb.RoundInfo{
			ID:       45,
			Topology: [][]byte{instance.GetID().Marshal()},
		},
	}

	err = ReceiveRoundTripPing(instance, msg)
	if err != nil {
		t.Errorf("ReceiveRoundTripPing returned an error: %+v", err)
	}

	if before == r.GetRTEnd().String() {
		t.Error("ReceiveRoundTripPing didn't update end time")
	}
}
