////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package io

import (
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/testkeys"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/server/internal/phase"
	"gitlab.com/elixxir/server/internal/round"
	"gitlab.com/elixxir/server/testUtil"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/crypto/large"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/utils"
	"testing"
)

// Happy path test
func TestTransmitStartSharePhase(t *testing.T) {
	roundID := id.Round(0)
	grp := cyclic.NewGroup(large.NewIntFromString(primeString, 16),
		large.NewInt(2))

	instance, nodeAddr := mockInstance(t, mockSharePhaseImpl)
	topology := connect.NewCircuit([]*id.ID{instance.GetID()})

	cert, _ := utils.ReadFile(testkeys.GetNodeCertPath())
	nodeHost, _ := connect.NewHost(instance.GetID(), nodeAddr, cert, connect.GetDefaultHostParams())
	topology.AddHost(nodeHost)
	_, err := instance.GetNetwork().AddHost(instance.GetID(), nodeAddr, cert, connect.GetDefaultHostParams())
	if err != nil {
		t.Errorf("Failed to add host to instance: %v", err)
	}

	response := phase.NewResponse(phase.ResponseDefinition{
		PhaseAtSource:  phase.PrecompGeneration,
		ExpectedStates: []phase.State{phase.Active},
		PhaseToExecute: phase.PrecompGeneration})

	p := testUtil.InitMockPhase(t)
	p.Ptype = phase.PrecompShare
	responseMap := make(phase.ResponseMap)
	responseMap[phase.PrecompShare.String()] = response

	rnd, err := round.New(grp, roundID, []phase.Phase{p}, responseMap, topology, topology.GetNodeAtIndex(0), 3, instance.GetRngStreamGen(), nil, "0.0.0.0", nil, nil, nil, nil)
	if err != nil {
		t.Error()
	}
	instance.GetRoundManager().AddRound(rnd)

	err = TransmitStartSharePhase(roundID, instance)
	if err != nil {
		t.Errorf("Failed to transmit: %+v", err)
	}
}

// Happy path test
func TestTransmitPhaseShare(t *testing.T) {
	roundID := id.Round(0)
	grp := cyclic.NewGroup(large.NewIntFromString(primeString, 16),
		large.NewInt(2))

	instance, nodeAddr := mockInstance(t, mockSharePhaseImpl)
	topology := connect.NewCircuit([]*id.ID{instance.GetID()})

	cert, _ := utils.ReadFile(testkeys.GetNodeCertPath())
	nodeHost, _ := connect.NewHost(instance.GetID(), nodeAddr, cert, connect.GetDefaultHostParams())
	topology.AddHost(nodeHost)
	_, err := instance.GetNetwork().AddHost(instance.GetID(), nodeAddr, cert, connect.GetDefaultHostParams())
	if err != nil {
		t.Errorf("Failed to add host to instance: %v", err)
	}

	response := phase.NewResponse(phase.ResponseDefinition{
		PhaseAtSource:  phase.PrecompGeneration,
		ExpectedStates: []phase.State{phase.Active},
		PhaseToExecute: phase.PrecompGeneration})

	p := testUtil.InitMockPhase(t)
	p.Ptype = phase.PrecompShare
	responseMap := make(phase.ResponseMap)
	responseMap[phase.PrecompShare.String()] = response

	rnd, err := round.New(grp, roundID, []phase.Phase{p}, responseMap, topology, topology.GetNodeAtIndex(0), 3, instance.GetRngStreamGen(), nil, "0.0.0.0", nil, nil, nil, nil)
	if err != nil {
		t.Error()
	}
	instance.GetRoundManager().AddRound(rnd)

	// Non-nil piece transmission
	msg := &pb.SharePiece{
		Piece:        grp.GetG().Bytes(),
		Participants: make([][]byte, 0),
		RoundID:      uint64(rnd.GetID()),
	}
	piece, err := generateShare(msg, grp, rnd, instance)
	if err != nil {
		t.Errorf("Could not generate a mock share: %v", err)
	}
	err = TransmitPhaseShare(instance, rnd, piece)
	if err != nil {
		t.Errorf("Failed to transmit: %+v", err)
	}

}
