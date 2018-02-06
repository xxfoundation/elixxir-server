package io

import (
	pb "gitlab.com/privategrity/comms/mixmessages"
	"gitlab.com/privategrity/server/services"
)

// Blank struct for implementing services.BatchTransmission
type RealtimePermuteHandler struct{}

// ReceptionHandler for RealtimePermuteMessages
func (s ServerImpl) RealtimePermute(input *pb.RealtimePermuteMessage) {}

// TransmissionHandler for RealtimePermuteMessages
func (h RealtimePermuteHandler) Handler(
	roundId string, batchSize uint64, slots []*services.Slot) {
}
