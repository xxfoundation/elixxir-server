package io

import (
	pb "gitlab.com/privategrity/comms/mixmessages"
	"gitlab.com/privategrity/server/services"
)

// Blank struct for implementing services.BatchTransmission
type RealtimeEncryptHandler struct{}

// ReceptionHandler for RealtimeEncryptMessages
func (s ServerImpl) RealtimeEncrypt(input *pb.RealtimeEncryptMessage) {}

// TransmissionHandler for RealtimeEncryptMessages
func (h RealtimeEncryptHandler) Handler(
	roundId string, batchSize uint64, slots []*services.Slot) {
}
