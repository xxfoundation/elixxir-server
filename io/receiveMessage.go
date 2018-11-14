////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package io

import (
	jww "github.com/spf13/jwalterweatherman"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/id"
	"gitlab.com/elixxir/server/cryptops/realtime"
	"gitlab.com/elixxir/server/globals"
)

type ReceiveMessageHandler struct{}

// Reception handler for ReceiveMessageFromClient
func (s ServerImpl) ReceiveMessageFromClient(msg *pb.CmixMessage) {
	recipientID := cyclic.NewIntFromBytes(msg.RecipientID)
	messagePayload := cyclic.NewIntFromBytes(msg.MessagePayload)

	jww.DEBUG.Printf("Received message from client: %v",
		messagePayload.Text(10))

	// Verify message fields are within the global cyclic group
	if globals.Grp.Inside(recipientID) && globals.Grp.Inside(messagePayload) {
		// Convert message to a Slot
		userId := new(id.UserID).SetBytes(msg.SenderID)
		inputMsg := realtime.Slot{
			Slot:               0, // Set in RunRealTime() in node/node.go
			CurrentID:          userId,
			Message:            messagePayload,
			EncryptedRecipient: recipientID,
			CurrentKey:         cyclic.NewMaxInt(),
			Salt:               msg.Salt,
		}
		MessageCh <- &inputMsg

	} else {
		jww.ERROR.Printf("Received message is not in the group: MsgPayload %v "+
			"RecipientID %v", messagePayload.Text(10), recipientID.Text(10))
	}
}
