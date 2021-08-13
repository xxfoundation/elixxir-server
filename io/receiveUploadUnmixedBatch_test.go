///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package io

import (
	"context"
	"encoding/base64"
	"github.com/pkg/errors"
	"git.xx.network/elixxir/comms/mixmessages"
	"git.xx.network/elixxir/primitives/current"
	"git.xx.network/elixxir/server/graphs/realtime"
	"git.xx.network/elixxir/server/internal/phase"
	"git.xx.network/elixxir/server/internal/round"
	"git.xx.network/elixxir/server/services"
	"git.xx.network/elixxir/server/storage"
	"git.xx.network/elixxir/server/testUtil"
	"git.xx.network/xx_network/comms/connect"
	"git.xx.network/xx_network/comms/messages"
	"git.xx.network/xx_network/primitives/id"
	"google.golang.org/grpc/metadata"
	"io"
	"runtime"
	"strings"
	"testing"
	"time"
)

/* MockStreamUnmixedBatchServer */
type MockStreamUnmixedBatchServer struct {
	batch                           *mixmessages.Batch
	mockStreamUnmixedBatchSlotIndex int
}

var mockUploadBatchIndex int

func (stream MockStreamUnmixedBatchServer) SendAndClose(*messages.Ack) error {
	if len(stream.batch.Slots) == mockUploadBatchIndex {
		return nil
	}
	return errors.Errorf("stream closed without all slots being received."+
		"\n\tMockStreamSlotIndex: %v\n\tstream.batch.slots: %v",
		stream.mockStreamUnmixedBatchSlotIndex, len(stream.batch.Slots))
}

func (stream MockStreamUnmixedBatchServer) Recv() (*mixmessages.Slot, error) {
	if mockUploadBatchIndex >= len(stream.batch.Slots) {
		return nil, io.EOF
	}
	slot := stream.batch.Slots[mockUploadBatchIndex]
	mockUploadBatchIndex++
	return slot, nil
}

func (MockStreamUnmixedBatchServer) SetHeader(metadata.MD) error {
	return nil
}

func (MockStreamUnmixedBatchServer) SendHeader(metadata.MD) error {
	return nil
}

func (MockStreamUnmixedBatchServer) SetTrailer(metadata.MD) {
}

func (stream MockStreamUnmixedBatchServer) Context() context.Context {
	// Create mock batch info from mock batch
	mockBatch := stream.batch
	mockBatchInfo := mixmessages.BatchInfo{
		Round:     mockBatch.Round,
		FromPhase: mockBatch.FromPhase,
		BatchSize: uint32(len(mockBatch.Slots)),
	}

	// Create an incoming context from batch info metadata
	ctx, _ := context.WithCancel(context.Background())

	m := make(map[string]string)
	m[mixmessages.UnmixedBatchHeader] = base64.StdEncoding.EncodeToString([]byte(mockBatchInfo.String()))

	md := metadata.New(m)
	ctx = metadata.NewIncomingContext(ctx, md)

	return ctx
}

func (MockStreamUnmixedBatchServer) SendMsg(m interface{}) error {
	return nil
}

func (MockStreamUnmixedBatchServer) RecvMsg(m interface{}) error {
	return nil
}

var postPhase = func(p phase.Phase, batch *mixmessages.Batch) error {
	return nil
}

func TestReceivePostNewBatch_Errors(t *testing.T) {
	// This round should be at a state where its precomp is complete.
	// So, we might want more than one phase,
	// since it's at a boundary between phases.
	instance, topology, grp := createMockInstance(t, 0, current.REALTIME)

	const batchSize = 1
	const roundID = 2

	// Does the mockPhase move through states?
	precompReveal := testUtil.InitMockPhase(t)
	precompReveal.Ptype = phase.PrecompReveal
	precompReveal.SetState(t, phase.Active)
	realDecrypt := testUtil.InitMockPhase(t)
	realDecrypt.Ptype = phase.RealDecrypt
	realDecrypt.SetState(t, phase.Active)

	tagKey := realDecrypt.Ptype.String()
	responseMap := make(phase.ResponseMap)
	responseMap[tagKey] = phase.NewResponse(phase.ResponseDefinition{
		PhaseAtSource:  realDecrypt.GetType(),
		PhaseToExecute: realDecrypt.GetType(),
		ExpectedStates: []phase.State{phase.Active},
	})

	// Well, this round needs to at least be on the precomp queue?
	// If it's not on the precomp queue,
	// that would let us test the error being returned.
	r, err := round.New(grp, instance.GetStorage(), roundID,
		[]phase.Phase{precompReveal, realDecrypt}, responseMap, topology,
		topology.GetNodeAtIndex(0), batchSize, instance.GetRngStreamGen(),
		nil, "0.0.0.0", nil, nil)
	if err != nil {
		t.Errorf("Failed to create new round: %+v", err)
	}
	instance.GetRoundManager().AddRound(r)

	var nodeIds [][]byte
	tempTopology := BuildMockNodeIDs(5, t)
	for _, tempId := range tempTopology {
		nodeIds = append(nodeIds, tempId.Marshal())
	}

	params := connect.GetDefaultHostParams()
	params.AuthEnabled = false
	h, _ := connect.NewHost(instance.GetGateway(), "test", nil, params)
	auth := &connect.Auth{
		IsAuthenticated: true,
		Sender:          h,
	}

	// Build a mock mockBatch to receive
	mockBatch := &mixmessages.Batch{
		Round: &mixmessages.RoundInfo{
			ID:       roundID + 10,
			Topology: nodeIds,
		},
		FromPhase: int32(phase.RealDecrypt),
		Slots: []*mixmessages.Slot{
			{
				// Do the fields need to be populated?
				SenderID: nil,
				PayloadA: nil,
				PayloadB: nil,
				Salt:     nil,
				KMACs:    nil,
			},
		},
	}
	for i := uint32(0); i < batchSize; i++ {
		mockBatch.Slots = append(mockBatch.Slots,
			&mixmessages.Slot{
				Index:    i,
				PayloadA: []byte{byte(i)},
			})
	}

	mockStreamServer := MockStreamUnmixedBatchServer{
		batch:                           mockBatch,
		mockStreamUnmixedBatchSlotIndex: 0,
	}

	err = ReceiveUploadUnmixedBatchStream(instance, mockStreamServer, postPhase, auth)
	if err == nil {
		t.Error("ReceiveUploadUnmixedBatchStream should have errored out if the round ID was not found")
	}

	// OK, let's put that round on the queue of completed precomps now,
	// which should cause the reception handler to function normally.
	// This should panic because the expected states aren't populated correctly,
	// so the realtime can't continue to be processed.
	defer func() {
		panicResult := recover()
		panicString := panicResult.(string)
		if panicString == "" {
			t.Error("There was no panicked error from the HandleIncomingComm" +
				" call")
		}
	}()

	mockBatch = &mixmessages.Batch{
		Round: &mixmessages.RoundInfo{
			ID: roundID,
		},
		FromPhase: int32(phase.RealDecrypt),
		Slots:     []*mixmessages.Slot{},
	}

	h, _ = connect.NewHost(instance.GetGateway(), "test", nil, params)
	auth = &connect.Auth{
		IsAuthenticated: true,
		Sender:          h,
	}

	mockUploadBatchIndex = 0
	mockStreamServer = MockStreamUnmixedBatchServer{
		batch:                           mockBatch,
		mockStreamUnmixedBatchSlotIndex: 0,
	}

	err = ReceiveUploadUnmixedBatchStream(instance, mockStreamServer, postPhase, auth)

}

// Test error case in which sender of postnewbatch is not authenticated
func TestReceivePostNewBatch_AuthError(t *testing.T) {
	instance, _ := mockServerInstance(t, current.REALTIME)

	const roundID = 2

	mockBatch := &mixmessages.Batch{
		Round: &mixmessages.RoundInfo{
			ID: roundID,
		},
		FromPhase: int32(phase.RealDecrypt),
		Slots: []*mixmessages.Slot{
			{
				// Do the fields need to be populated?
				SenderID: nil,
				PayloadA: nil,
				PayloadB: nil,
				Salt:     nil,
				KMACs:    nil,
			},
		},
	}
	batchSize := uint32(32)
	// Build a mock mockBatch to receive
	mockPhase := testUtil.InitMockPhase(t)
	for i := uint32(0); i < batchSize; i++ {
		mockBatch.Slots = append(mockBatch.Slots,
			&mixmessages.Slot{
				Index:    i,
				PayloadA: []byte{byte(i)},
			})
	}

	mockBatch.FromPhase = int32(mockPhase.GetType())
	mockBatch.Round = &mixmessages.RoundInfo{ID: uint64(roundID)}

	mockStreamServer := MockStreamUnmixedBatchServer{
		batch: mockBatch,
	}

	params := connect.GetDefaultHostParams()
	params.AuthEnabled = false
	h, _ := connect.NewHost(instance.GetGateway(), "test", nil, params)
	auth := &connect.Auth{
		IsAuthenticated: false,
		Sender:          h,
	}

	err := ReceiveUploadUnmixedBatchStream(instance, mockStreamServer, postPhase, auth)

	if err == nil {
		t.Error("Did not receive expected error")
		return
	}

	if !strings.Contains(err.Error(), "Failed to authenticate") {
		t.Error("Did not receive expected authentication error")
	}
}

// Test error case in which the sender of postnewbatch is not who we expect
func TestReceivePostNewBatch_BadSender(t *testing.T) {
	instance, _ := mockServerInstance(t, current.REALTIME)

	const roundID = 2

	// Build a mock mockBatch to receive
	mockBatch := &mixmessages.Batch{
		Round: &mixmessages.RoundInfo{
			ID: roundID + 10,
		},
		FromPhase: int32(phase.RealDecrypt),
		Slots: []*mixmessages.Slot{
			{
				// Do the fields need to be populated?
				SenderID: nil,
				PayloadA: nil,
				PayloadB: nil,
				Salt:     nil,
				KMACs:    nil,
			},
		},
	}
	mockPhase := testUtil.InitMockPhase(t)
	for i := uint32(0); i < 32; i++ {
		mockBatch.Slots = append(mockBatch.Slots,
			&mixmessages.Slot{
				Index:    i,
				PayloadA: []byte{byte(i)},
			})
	}

	mockBatch.FromPhase = int32(mockPhase.GetType())
	mockBatch.Round = &mixmessages.RoundInfo{ID: uint64(roundID)}

	mockStreamServer := MockStreamUnmixedBatchServer{
		batch: mockBatch,
	}

	newID := id.NewIdFromString("test", id.Node, t)

	params := connect.GetDefaultHostParams()
	params.AuthEnabled = false
	h, _ := connect.NewHost(newID, "test", nil, params)
	auth := &connect.Auth{
		IsAuthenticated: true,
		Sender:          h,
	}

	err := ReceiveUploadUnmixedBatchStream(instance, mockStreamServer, postPhase, auth)

	if err == nil {
		t.Error("Did not receive expected error")
		return
	}

	if !strings.Contains(err.Error(), "Failed to authenticate") {
		t.Error("Did not receive expected authentication error")
	}
}

// Tests the happy path of ReceiveUploadUnmixedBatchStream, demonstrating that it can start
// realtime processing with a new batch from the gateway.
// Note: In this case, the happy path includes an error from one of the slots
// that has cryptographically incorrect data.
func TestReceivePostNewBatch(t *testing.T) {
	instance, topology, grp := createMockInstance(t, 0, current.REALTIME)
	registry := instance.GetStorage()

	// Make and register a user
	sender := &storage.Client{
		Id:             id.NewIdFromString("test", id.User, &testing.T{}).Marshal(),
		DhKey:          nil,
		PublicKey:      nil,
		Nonce:          nil,
		NonceTimestamp: time.Time{},
		IsRegistered:   false,
	}
	_ = registry.UpsertClient(sender)

	const batchSize = 1
	const roundID = 2

	gg := services.NewGraphGenerator(4, uint8(runtime.NumCPU()),
		1, 1.0)

	realDecrypt := phase.New(phase.Definition{
		Graph: realtime.InitDecryptGraph(gg),
		Type:  phase.RealDecrypt,
		TransmissionHandler: func(roundID id.Round, instance phase.GenericInstance, getChunk phase.GetChunk,
			getMessage phase.GetMessage) error {
			return nil
		},
		Timeout:        5 * time.Second,
		DoVerification: false,
	})

	tagKey := realDecrypt.GetType().String()
	responseMap := make(phase.ResponseMap)
	responseMap[tagKey] = phase.NewResponse(phase.ResponseDefinition{
		PhaseAtSource:  realDecrypt.GetType(),
		ExpectedStates: []phase.State{phase.Active},
		PhaseToExecute: realDecrypt.GetType(),
	})

	// We need this round to be on the precomp queue
	r, err := round.New(grp, instance.GetStorage(), roundID,
		[]phase.Phase{realDecrypt}, responseMap, topology,
		topology.GetNodeAtIndex(0), batchSize, instance.GetRngStreamGen(),
		nil, "0.0.0.0", nil, nil)
	if err != nil {
		t.Errorf("Failed to create new round: %+v", err)
	}
	instance.GetRoundManager().AddRound(r)

	var nodeIds [][]byte
	tempTopology := BuildMockNodeIDs(5, t)
	for _, tempId := range tempTopology {
		nodeIds = append(nodeIds, tempId.Marshal())
	}

	params := connect.GetDefaultHostParams()
	params.AuthEnabled = false
	h, _ := connect.NewHost(instance.GetGateway(), "test", nil, params)
	auth := &connect.Auth{
		IsAuthenticated: true,
		Sender:          h,
	}

	// Build a mock mockBatch to receive
	mockBatch := &mixmessages.Batch{
		Round: &mixmessages.RoundInfo{
			ID:       roundID,
			Topology: nodeIds,
		},
		FromPhase: int32(phase.RealDecrypt),
		Slots:     []*mixmessages.Slot{},
	}
	for i := uint32(0); i < batchSize; i++ {
		mockBatch.Slots = append(mockBatch.Slots,
			&mixmessages.Slot{
				Index:    i,
				PayloadA: []byte{byte(i)},
			})
	}

	mockStreamServer := MockStreamUnmixedBatchServer{
		batch:                           mockBatch,
		mockStreamUnmixedBatchSlotIndex: 0,
	}

	// Actually, this should return an error because the batch has a malformed
	// slot in it, so once we implement per-slot errors we can test all the
	// realtime decrypt error cases from this reception handler if we want
	err = ReceiveUploadUnmixedBatchStream(instance, mockStreamServer, postPhase, auth)
	if err != nil {
		t.Error(err)
	}

	// We verify that the Realtime Decrypt phase has been enqueued
	if !realDecrypt.IsQueued() {
		t.Errorf("Realtime decrypt is not queued")
	}
}
