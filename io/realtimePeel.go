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
	"time"
)

// Blank struct for implementing services.BatchTransmission
type RealtimePeelHandler struct{}

// TransmissionHandler for RealtimePeelMessages
func (h RealtimePeelHandler) Handler(
	roundId string, batchSize uint64, slots []*services.Slot) {
	startTime := time.Now()
	jww.INFO.Printf("Starting RealtimePeel.Handler(RoundId: %s) at %s",
		roundId, startTime.Format(time.RFC3339))

	// Retrieve the EncryptedMessage
	round := globals.GlobalRoundMap.GetRound(roundId)
	if round == nil {
		jww.INFO.Printf("skipping round %s, because it's dead", roundId)
		return
	}
	messageBatch := make([]*pb.CmixMessage, 0)
	for i := uint64(0); i < batchSize; i++ {
		slot := (*slots[i]).(*realtime.RealtimeSlot)
		if !round.MIC_Verification[slot.Slot] {
			jww.DEBUG.Printf("Message %v corrupted, not queued", slot.SlotID())
		} else {
			jww.DEBUG.Printf("EncryptedMessage Result: %s",
				slot.Message.Text(10))
			user, err := globals.Users.GetUser(slot.CurrentID)

			if err != nil {
				jww.ERROR.Printf("Could not store message for invalid"+
					" user: %v", slot.CurrentID)
				continue
			}

			pbCmixMessage := pb.CmixMessage{
				SenderID:       uint64(0), // Currently zero this field
				MessagePayload: slot.Message.LeftpadBytes(512),
				RecipientID:    make([]byte, 0), // Currently zero this field
			}
			messageBatch = append(messageBatch, &pbCmixMessage)

			for !addMessageToBuffer(user, &pbCmixMessage) {
				<-user.MessageBuffer
				if user.ID != uint64(35) {
					jww.WARN.Printf("Message dropped for user %v because"+
						" message buffer is full", user.ID)
				}
			}
		}
	}
	if globals.GatewayAddress != "" {
		node.SendReceiveBatch(globals.GatewayAddress, messageBatch)
	}

	globals.GlobalRoundMap.SetPhase(roundId, globals.REAL_COMPLETE)

	endTime := time.Now()
	jww.INFO.Printf("Finished RealtimePeel.Handler(RoundId: %s) in %d ms",
		roundId, (endTime.Sub(startTime))/time.Millisecond)
	jww.INFO.Printf("Realtime for Round %s Finished at %s!", roundId, endTime)
}

func addMessageToBuffer(user *globals.User, pbCmixMessage *pb.CmixMessage) bool {
	select {
	case user.MessageBuffer <- pbCmixMessage:
		return true
	default:
		return false
	}
}
