package io

import (
	pb "gitlab.com/privategrity/comms/mixmessages"
	"gitlab.com/privategrity/server/services"
)

// Blank struct for implementing services.BatchTransmission
type PrecompEncryptHandler struct{}

// ReceptionHandler for PrecompEncryptMessages
func (s ServerImpl) PrecompEncrypt(input *pb.PrecompEncryptMessage) {}

// TransmissionHandler for PrecompEncryptMessages
func (h PrecompEncryptHandler) Handler(
	roundId string, batchSize uint64, slots []*services.Slot) {
}
