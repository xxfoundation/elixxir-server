package io

import (
	jww "github.com/spf13/jwalterweatherman"
	pb "gitlab.com/privategrity/comms/mixmessages"
	"gitlab.com/privategrity/comms/mixserver/message"
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/server/cryptops/precomputation"
	"gitlab.com/privategrity/server/globals"
	"gitlab.com/privategrity/server/services"
)

// Blank struct for implementing services.BatchTransmission
type PrecompShareHandler struct{}

// ReceptionHandler for PrecompShareMessages
func (s ServerImpl) PrecompShare(input *pb.PrecompShareMessage) {
	chIn := s.GetChannel(input.RoundID, globals.PRECOMP_SHARE)
	// Iterate through the Slots in the PrecompShareMessage
	for i := 0; i < len(input.Slots); i++ {
		// Convert input message to equivalent SlotShare
		in := input.Slots[i]
		var slot services.Slot = &precomputation.SlotShare{
			Slot: in.Slot,
			PartialRoundPublicCypherKey: cyclic.NewIntFromBytes(
				in.PartialRoundPublicCypherKey),
		}
		// Pass slot as input to Share's channel
		chIn <- &slot
	}
}

// TransmissionHandler for PrecompShareMessages
func (h PrecompShareHandler) Handler(
	roundId string, batchSize uint64, slots []*services.Slot) {
	// Create the PrecompShareMessage
	msg := &pb.PrecompShareMessage{
		RoundID: roundId,
		Slots:   make([]*pb.PrecompShareSlot, batchSize),
	}

	// Iterate over the output channel
	for i := uint64(0); i < batchSize; i++ {
		// Type assert Slot to SlotShare
		out := (*slots[i]).(*precomputation.SlotShare)
		// Convert to PrecompShareSlot
		msgSlot := &pb.PrecompShareSlot{
			Slot: out.Slot,
			PartialRoundPublicCypherKey: out.
				PartialRoundPublicCypherKey.Bytes(),
		}

		// Put it into the slice
		msg.Slots[i] = msgSlot
	}
	// Send the completed PrecompShareMessage
	jww.INFO.Printf("Sending PrecompShare Message to %v...", NextServer)
	message.SendPrecompShare(NextServer, msg)
}
