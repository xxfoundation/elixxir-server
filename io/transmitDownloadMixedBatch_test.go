///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package io

import (
	"context"
	"github.com/pkg/errors"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/server/internal/round"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/comms/messages"
	"gitlab.com/xx_network/primitives/id"
	"google.golang.org/grpc/metadata"
	"io"
	"testing"
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

	mockStream := MockStreamMixedBatchServer{}

	ready := &pb.BatchReady{RoundId: uint64(rid)}
	err = DownloadMixedBatch(&instance, ready, mockStream, auth)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

}

/* MockStreamUnmixedBatchServer */
type MockStreamMixedBatchServer struct {
	batch                           *pb.Batch
	mockStreamUnmixedBatchSlotIndex int
}

func (stream MockStreamMixedBatchServer) Send(slot *pb.Slot) error {
	return nil
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

	// Create an incoming context from batch info metadata
	ctx, _ := context.WithCancel(context.Background())

	return ctx
}

func (MockStreamMixedBatchServer) SendMsg(m interface{}) error {
	return nil
}

func (MockStreamMixedBatchServer) RecvMsg(m interface{}) error {
	return nil
}
