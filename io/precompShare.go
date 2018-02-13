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
	jww.INFO.Printf("Received PrecompShare Message %v...", input.RoundID)
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

// Transition to PrecompDecrypt phase on the last node
func precompShareLastNode(roundId string, batchSize uint64,
	input *pb.PrecompShareMessage) {
	// For each node, set CypherPublicKey to
	// shareResult.PartialRoundPublicCypherKey
	jww.INFO.Println("Setting node Public Keys...")
	for i := range Servers {
		message.SetPublicKey(Servers[i], &pb.PublicKeyMessage{
			RoundID:   input.RoundID,
			PublicKey: input.Slots[0].PartialRoundPublicCypherKey,
		})
	}

	// Kick off the PrecompDecrypt phase
	jww.INFO.Println("Beginning PrecompDecrypt Phase...")

	// Create the PrecompDecryptMessage
	msg := &pb.PrecompDecryptMessage{
		RoundID: roundId,
		Slots:   make([]*pb.PrecompDecryptSlot, batchSize),
	}

	// Iterate over the input slots
	for i := uint64(0); i < batchSize; i++ {
		// Convert to PrecompDecryptSlot
		msgSlot := &pb.PrecompDecryptSlot{
			Slot:                         uint64(i),
			EncryptedMessageKeys:         cyclic.NewInt(1).Bytes(),
			PartialMessageCypherText:     cyclic.NewInt(1).Bytes(),
			EncryptedRecipientIDKeys:     cyclic.NewInt(1).Bytes(),
			PartialRecipientIDCypherText: cyclic.NewInt(1).Bytes(),
		}
		msg.Slots[i] = msgSlot
	}

	// Send first PrecompDecrypt Message
	jww.INFO.Printf("Sending PrecompDecrypt Message to %v...", NextServer)
	message.SendPrecompDecrypt(NextServer, msg)
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

	// Returns whether this is the first time Share is being run TODO Something better
	IsFirstRun := (*slots[0]).(*precomputation.SlotShare).PartialRoundPublicCypherKey.Cmp(globals.Grp.G) == 0
	if IsLastNode && !IsFirstRun {
		// Transition to PrecompDecrypt phase
		// if we are last node and this isn't the first run
		precompShareLastNode(roundId, batchSize, msg)
	} else {
		// Send the completed PrecompShareMessage
		jww.INFO.Printf("Sending PrecompShare Message to %v...", NextServer)
		message.SendPrecompShare(NextServer, msg)
	}
}
