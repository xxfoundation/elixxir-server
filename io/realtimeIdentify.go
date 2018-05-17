////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package io

import (
	jww "github.com/spf13/jwalterweatherman"
	pb "gitlab.com/privategrity/comms/mixmessages"
	"gitlab.com/privategrity/comms/node"
	"gitlab.com/privategrity/server/cryptops/realtime"
	"gitlab.com/privategrity/server/globals"
	"gitlab.com/privategrity/server/services"
	"strconv"
	"time"
)

// Blank struct for implementing services.BatchTransmission
type RealtimeIdentifyHandler struct{}

// TransmissionHandler for RealtimeIdentifyMessages
func (h RealtimeIdentifyHandler) Handler(
	roundId string, batchSize uint64, slots []*services.Slot) {
	startTime := time.Now()
	jww.INFO.Printf("Starting RealtimeIdentify.Handler(RoundId: %s) at %s",
		roundId, startTime.Format(time.RFC3339))

	// Create the RealtimeEncryptMessage
	msg := &pb.RealtimeEncryptMessage{
		RoundID: roundId,
		LastOp:  int32(globals.REAL_IDENTIFY),
		Slots:   make([]*pb.RealtimeEncryptSlot, batchSize),
	}

	// Get round
	round := globals.GlobalRoundMap.GetRound(roundId)

	// Iterate over the input slots
	for i := range slots {
		out := (*slots[i]).(*realtime.RealtimeSlot)
		// Convert to RealtimeEncryptSlot
		rId, _ := strconv.ParseUint(out.EncryptedRecipient.Text(10), 10, 64)
		msgSlot := &pb.RealtimeEncryptSlot{
			Slot:             out.Slot,
			RecipientID:      rId,
			EncryptedMessage: round.LastNode.EncryptedMessage[i].Bytes(),
		}

		// Append the RealtimeEncryptSlot to the RealtimeEncryptMessage
		msg.Slots[i] = msgSlot
	}

	// Advance internal state to the next phase
	globals.GlobalRoundMap.SetPhase(roundId, globals.REAL_ENCRYPT)

	// Send the first RealtimeEncrypt Message
	sendTime := time.Now()
	jww.INFO.Printf("Sending RealtimeEncrypt Messages to %v at %s",
		NextServer, sendTime.Format(time.RFC3339))
	node.SendRealtimeEncrypt(NextServer, msg)

	endTime := time.Now()
	jww.INFO.Printf("Finished RealtimeIdentify.Handler(RoundId: %s) in %d ms",
		roundId, (endTime.Sub(startTime))/time.Millisecond)
}
