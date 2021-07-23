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
	"gitlab.com/elixxir/comms/gateway"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/testkeys"
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/server/internal"
	"gitlab.com/elixxir/server/internal/measure"
	"gitlab.com/elixxir/server/internal/round"
	"gitlab.com/elixxir/server/internal/state"
	"gitlab.com/elixxir/server/testUtil"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/comms/gossip"
	"gitlab.com/xx_network/comms/messages"
	"gitlab.com/xx_network/primitives/id"
	ndf2 "gitlab.com/xx_network/primitives/ndf"
	"google.golang.org/grpc/metadata"
	"io"
	"reflect"
	"testing"
	"time"
)

// Happy path test
func TestStartDownloadMixedBatch(t *testing.T) {
	instance, _, _, _ := setupTests(t, current.REALTIME)

	dr := round.NewDummyRound(0, 10, t)
	instance.GetRoundManager().AddRound(dr)

	// Create host and auth
	params := connect.GetDefaultHostParams()
	params.AuthEnabled = false
	h, _ := connect.NewHost(instance.GetGateway(), testGatewayAddress, nil, params)
	auth := &connect.Auth{
		IsAuthenticated: true,
		Sender:          h,
	}

	rid := id.Round(32)
	cr := &round.CompletedRound{RoundID: rid}
	err := instance.AddCompletedBatch(cr)
	if err != nil {
		t.Fatalf("Could not send completed round ot channel: %v", err)
	}

	ready := &pb.BatchReady{RoundId: uint64(rid)}
	err = StartDownloadMixedBatch(&instance, ready, auth)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	received, err := instance.GetCompletedBatchSignal().Receive()
	if err != nil {
		t.Fatalf("Could not receive batch from channel")
	}

	if !reflect.DeepEqual(received, cr) {
		t.Fatalf("Expected completed round did not received."+
			"\n\tExpected: %v"+
			"\n\tReceived: %v", cr, received)
	}

}

func TestTransmitStreamDownloadBatch(t *testing.T) {
	//Get a new ndf
	testNdf, err := ndf2.Unmarshal(testUtil.ExampleNDF)
	if err != nil {
		t.Error("Failed to decode ndf")
	}

	// Since no deep copy of ndf exists we create a new object entirely for second ndf that
	// We use to test against
	test2Ndf, err := ndf2.Unmarshal(testUtil.ExampleNDF)
	if err != nil {
		t.Fatal("Failed to decode ndf 2")
	}

	// Change the time of the ndf so we can generate a different hash for use in comparisons
	test2Ndf.Timestamp = time.Now()

	ourGateway := internal.GW{
		ID:      &id.TempGateway,
		TlsCert: nil,
		Address: testGatewayAddress,
	}

	// We need to create a server.Definition so we can create a server instance.
	nid := internal.GenerateId(t)
	def := internal.Definition{
		ID:              nid,
		ResourceMonitor: &measure.ResourceMonitor{},
		FullNDF:         testNdf,
		PartialNDF:      testNdf,
		Gateway:         ourGateway,
		Flags:           internal.Flags{OverrideInternalIP: "0.0.0.0"},
		DevMode:         true,
	}
	def.Gateway.ID = def.ID.DeepCopy()
	def.Gateway.ID.SetType(id.Gateway)

	// Here we create a server instance so that we can test the poll ndf.
	m := state.NewTestMachine(dummyStates, current.REALTIME, t)

	instance, err := internal.CreateServerInstance(&def, NewImplementation, m, "1.1.0")
	if err != nil {
		t.Fatalf("failed to create server Instance")
	}

	dr := round.NewDummyRound(0, 10, t)
	instance.GetRoundManager().AddRound(dr)

	// Create host and auth
	params := connect.GetDefaultHostParams()
	params.AuthEnabled = false

	rid := id.Round(32)
	cr := &round.CompletedRound{
		RoundID: rid,
		Round:   make([]*pb.Slot, 0),
	}

	for i := uint32(0); i < 32; i++ {
		cr.Round = append(cr.Round,
			&pb.Slot{
				Index:    i,
				PayloadA: []byte{byte(i)},
			})
	}

	keyPath := testkeys.GetNodeKeyPath()
	keyData := testkeys.LoadFromPath(keyPath)
	certPath := testkeys.GetNodeCertPath()
	certData := testkeys.LoadFromPath(certPath)

	//receiverImpl := gateway.NewImplementation()
	//receiverImpl.Functions.DownloadMixedBatch = func(server pb.Gateway_DownloadMixedBatchServer, auth *connect.Auth) error {
	//	return nil
	//}

	newMockGwImpl := mockGatewayImpl{}

	gw := gateway.StartGateway(instance.GetGateway(), "0.0.0.0:5555",
		newMockGwImpl, certData, keyData, gossip.DefaultManagerFlags())
	defer gw.Shutdown()

	instance.GetNetwork().RemoveHost(instance.GetGateway())

	_, err = instance.GetNetwork().AddHost(instance.GetGateway(), "0.0.0.0:5555", certData, params)
	if err != nil {
		t.Fatalf("Could not add gateway as host: %v", err)
	}

	err = TransmitStreamDownloadBatch(instance, cr)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

}

/* MockStreamUnmixedBatchServer */
type MockStreamMixedBatchServer struct {
	batch                           *pb.Batch
	mockStreamUnmixedBatchSlotIndex int
}

var mockDownloadBatchIndex int

func (stream MockStreamMixedBatchServer) SendAndClose(*messages.Ack) error {
	if len(stream.batch.Slots) == mockDownloadBatchIndex {
		return nil
	}
	return errors.Errorf("stream closed without all slots being received."+
		"\n\tMockStreamSlotIndex: %v\n\tstream.batch.slots: %v",
		stream.mockStreamUnmixedBatchSlotIndex, len(stream.batch.Slots))
}

func (stream MockStreamMixedBatchServer) Recv() (*pb.Slot, error) {
	if mockDownloadBatchIndex >= len(stream.batch.Slots) {
		return nil, io.EOF
	}
	slot := stream.batch.Slots[mockDownloadBatchIndex]
	mockDownloadBatchIndex++
	return slot, nil
}

func (MockStreamMixedBatchServer) SetHeader(metadata.MD) error {
	return nil
}

func (MockStreamMixedBatchServer) SendHeader(metadata.MD) error {
	return nil
}

func (MockStreamMixedBatchServer) SetTrailer(metadata.MD) {
}

func (stream MockStreamMixedBatchServer) Context() context.Context {
	// Create mock batch info from mock batch
	mockBatch := stream.batch
	mockBatchInfo := pb.BatchInfo{
		Round:     mockBatch.Round,
		FromPhase: mockBatch.FromPhase,
		BatchSize: uint32(len(mockBatch.Slots)),
	}

	// Create an incoming context from batch info metadata
	ctx, _ := context.WithCancel(context.Background())

	m := make(map[string]string)
	m[pb.MixedBatchHeader] = base64.StdEncoding.EncodeToString([]byte(mockBatchInfo.String()))

	md := metadata.New(m)
	ctx = metadata.NewIncomingContext(ctx, md)

	return ctx
}

func (MockStreamMixedBatchServer) SendMsg(m interface{}) error {
	return nil
}

func (MockStreamMixedBatchServer) RecvMsg(m interface{}) error {
	return nil
}

type mockGatewayImpl struct{}

func (m mockGatewayImpl) PutMessage(message *pb.GatewaySlot) (*pb.GatewaySlotResponse, error) {
	panic("implement me")
}

func (m mockGatewayImpl) PutManyMessages(msgs *pb.GatewaySlots) (*pb.GatewaySlotResponse, error) {
	panic("implement me")
}

func (m mockGatewayImpl) RequestNonce(message *pb.NonceRequest) (*pb.Nonce, error) {
	panic("implement me")
}

func (m mockGatewayImpl) ConfirmNonce(message *pb.RequestRegistrationConfirmation) (*pb.RegistrationConfirmation, error) {
	panic("implement me")
}

func (m mockGatewayImpl) Poll(msg *pb.GatewayPoll) (*pb.GatewayPollResponse, error) {
	panic("implement me")
}

func (m mockGatewayImpl) RequestHistoricalRounds(msg *pb.HistoricalRounds) (*pb.HistoricalRoundsResponse, error) {
	panic("implement me")
}

func (m mockGatewayImpl) RequestMessages(msg *pb.GetMessages) (*pb.GetMessagesResponse, error) {
	panic("implement me")
}

func (m mockGatewayImpl) ShareMessages(msg *pb.RoundMessages, auth *connect.Auth) error {
	panic("implement me")
}

func (m mockGatewayImpl) DownloadMixedBatch(server pb.Gateway_DownloadMixedBatchServer, auth *connect.Auth) error {
	var slots []*pb.Slot
	index := uint32(0)
	for {
		slot, err := server.Recv()
		// If we are at end of receiving
		// send ack and finish
		if err == io.EOF {
			ack := messages.Ack{
				Error: "",
			}

			batchInfo, err := gateway.GetMixedBatchStreamHeader(server)
			if err != nil {
				return err
			}

			// Create batch using batch info header
			// and temporary slot buffer contents
			receivedBatch = &pb.Batch{
				Round:     batchInfo.Round,
				FromPhase: batchInfo.FromPhase,
				Slots:     slots,
			}

			err = server.SendAndClose(&ack)
			return err
		}

		// If we have another error, return err
		if err != nil {
			return err
		}

		// Store slot received into temporary buffer
		slots = append(slots, slot)

		index++
	}

	return nil
}
