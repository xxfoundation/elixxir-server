////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package io

import (
	"testing"
	pb "gitlab.com/privategrity/comms/mixmessages"
	"gitlab.com/privategrity/comms/mixclient"
	"gitlab.com/privategrity/crypto/format"
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/server/globals"
	"gitlab.com/privategrity/server/cryptops/realtime"
	"gitlab.com/privategrity/server/benchmark"
)

// Test that we can receive an unencrypted message on the server
func TestServerImpl_ReceiveMessageFromClient(t *testing.T) {
	// Set up the necessary globals
	g := cyclic.NewGroup(cyclic.NewIntFromString(
		benchmark.PRIME,16), cyclic.NewInt(0), cyclic.NewInt(5),
		cyclic.NewRandom(cyclic.NewInt(0), cyclic.NewInt(1000)))
	globals.Grp = &g
	MessageCh = make(chan *realtime.RealtimeSlot)

	// Expected values
	userID := uint64(1)
	payload := "hey there, sailor. want to see my unencrypted message?"

	// Create an unencrypted message for testing
	message, err := format.NewMessage(userID, userID, payload)

	if err != nil {
		t.Errorf("Couldn't construct message: %s", err.Error())
	}
	messageSerial := message[0].SerializeMessage()

	// Send the message to the server itself
	_, err = mixclient.SendMessageToServer(NextServer,
		&pb.CmixMessage{message[0].GetSenderIDUint(),
			messageSerial.Payload.Bytes(), messageSerial.Recipient.Bytes()})
	if err != nil {
		t.Errorf("Couldn't send message to server: %s", err.Error())
	}
	receivedMessage := <-MessageCh

	// Verify the received message
	if receivedMessage.CurrentID != userID {
		t.Errorf("Received sender ID %v, expected %v",
			receivedMessage.CurrentID, userID)
	}
	result := format.DeserializeMessage(format.MessageSerial{receivedMessage.
		Message,
	receivedMessage.EncryptedRecipient})
	if result.GetSenderIDUint() != userID {
		t.Errorf("Received sender ID in bytes %v, expected %v",
			result.GetSenderIDUint(), userID)
	}
	if result.GetRecipientIDUint() != userID {
		t.Errorf("Received recipient ID %v, expected %v",
			result.GetRecipientIDUint(), userID)
	}
	if result.GetPayload() != payload {
		t.Errorf("Received payload message %v, expected %v",
			result.GetPayload(), payload)
	}
	// todo generate and verify mics
}
