package io

import (
	pb "gitlab.com/privategrity/comms/mixmessages"
	"gitlab.com/privategrity/server/services"
)

// Blank struct for implementing services.BatchTransmission
type RealtimeIdentifyHandler struct{}

// ReceptionHandler for RealtimeIdentifyMessages
func (s ServerImpl) RealtimeIdentify(input *pb.RealtimeIdentifyMessage) {}

// TransmissionHandler for RealtimeIdentifyMessages
func (h RealtimeIdentifyHandler) Handler(
	roundId string, batchSize uint64, slots []*services.Slot) {
}
