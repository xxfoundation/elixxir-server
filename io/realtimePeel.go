////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package io

import (
	jww "github.com/spf13/jwalterweatherman"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/server/cryptops/realtime"
	"gitlab.com/elixxir/server/globals"
	"gitlab.com/elixxir/server/services"
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

	elapsed := startTime.Sub(globals.GlobalRoundMap.GetRound(roundId).
		CryptopStartTimes[globals.REAL_PEEL])

	jww.DEBUG.Printf("RealtimePeel Crypto took %v ms for "+
		"RoundId %s", elapsed, roundId)

	messageBatch := make([]*pb.CmixMessage, 0)
	for i := uint64(0); i < batchSize; i++ {
		slot := (*slots[i]).(*realtime.Slot)
		if !round.MIC_Verification[slot.Slot] {
			jww.DEBUG.Printf("Message %v corrupted, not queued", slot.SlotID())
		} else {
			jww.DEBUG.Printf("EncryptedMessage Result: %s",
				slot.Message.Text(10))
			_, err := globals.Users.GetUser(slot.CurrentID)

			if err != nil {
				jww.ERROR.Printf("Could not store message for invalid"+
					" user: %v", slot.CurrentID)
				continue
			}

			pbCmixMessage := pb.CmixMessage{
				SenderID:       make([]byte, 0),
				MessagePayload: slot.Message.LeftpadBytes(512),
				AssociatedData: slot.AssociatedData.LeftpadBytes(512),
				Salt:           slot.Salt,
			}
			messageBatch = append(messageBatch, &pbCmixMessage)
		}
	}
	if globals.GatewayAddress != "" {
		jww.INFO.Printf("Sending completed batch to gateway %v", globals.GatewayAddress)
		node.SendReceiveBatch(globals.GatewayAddress, messageBatch)
	}

	globals.GlobalRoundMap.SetPhase(roundId, globals.REAL_COMPLETE)

	endTime := time.Now()
	jww.INFO.Printf("Finished RealtimePeel.Handler(RoundId: %s) in %d ms",
		roundId, (endTime.Sub(startTime))/time.Millisecond)
	jww.INFO.Printf("Realtime for Round %s Finished at %s!", roundId, endTime)
}
