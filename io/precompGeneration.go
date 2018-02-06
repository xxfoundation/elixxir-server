package io

import (
	pb "gitlab.com/privategrity/comms/mixmessages"
	"gitlab.com/privategrity/server/services"
)

// Blank struct for implementing services.BatchTransmission
type PrecompGenerationHandler struct{}

// ReceptionHandler for PrecompGenerationMessages
func (s ServerImpl) PrecompGeneration(input *pb.PrecompGenerationMessage) {}

// TransmissionHandler for PrecompGenerationMessages
func (h PrecompGenerationHandler) Handler(
	roundId string, batchSize uint64, slots []*services.Slot) {
}
