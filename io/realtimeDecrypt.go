package io

import (
	pb "gitlab.com/privategrity/comms/mixmessages"
	"gitlab.com/privategrity/server/services"
)

// Blank struct for implementing services.BatchTransmission
type RealtimeDecryptHandler struct{}

// ReceptionHandler for RealtimeDecryptMessages
func (s ServerImpl) RealtimeDecrypt(input *pb.RealtimeDecryptMessage) {}

// TransmissionHandler for RealtimeDecryptMessages
func (h RealtimeDecryptHandler) Handler(
	roundId string, batchSize uint64, slots []*services.Slot) {
}
