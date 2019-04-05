////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package io

import (
	jww "github.com/spf13/jwalterweatherman"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/large"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/cryptops/realtime"
	"gitlab.com/elixxir/server/globals"
)

type ReceiveMessageHandler struct{}

// Reception handler for ReceiveMessageFromClient
func ReceiveMessageFromClient(msg *pb.CmixMessage) {

	grp := globals.GetGroup()

	messagePayloadLarge := large.NewIntFromBytes(msg.MessagePayload)
	associatedDataLarge := large.NewIntFromBytes(msg.AssociatedData)

	//Check that the message is in the group and overwrite
	if !grp.Inside(messagePayloadLarge) || !grp.Inside(associatedDataLarge) {
		jww.ERROR.Printf("Message from client outside the group: %v %v",
			messagePayloadLarge.Text(10), associatedDataLarge.Text(10))
		associatedDataLarge.SetInt64(1)
		messagePayloadLarge.SetInt64(1)
	}

	messagePayload := grp.NewIntFromLargeInt(messagePayloadLarge)
	associatedData := grp.NewIntFromLargeInt(associatedDataLarge)

	jww.DEBUG.Printf("Received message from client: %v",
		messagePayload.Text(10))

	// Convert message to a Slot
	userId := new(id.User).SetBytes(msg.SenderID)
	inputMsg := realtime.Slot{
		Slot:           0, // Set in RunRealTime() in node/node.go
		CurrentID:      userId,
		Message:        messagePayload,
		AssociatedData: associatedData,
		CurrentKey:     globals.GetGroup().NewMaxInt(),
		Salt:           msg.Salt,
	}

	MessageCh <- &inputMsg

}
