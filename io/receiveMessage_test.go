////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package io

import (
	"bytes"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/cryptops/realtime"
	"gitlab.com/elixxir/server/globals"
	"testing"
)

var PRIME = "FFFFFFFFFFFFFFFFC90FDAA22168C234C4C6628B80DC1CD1" +
	"29024E088A67CC74020BBEA63B139B22514A08798E3404DD" +
	"EF9519B3CD3A431B302B0A6DF25F14374FE1356D6D51C245" +
	"E485B576625E7EC6F44C42E9A637ED6B0BFF5CB6F406B7ED" +
	"EE386BFB5A899FA5AE9F24117C4B1FE649286651ECE45B3D" +
	"C2007CB8A163BF0598DA48361C55D39A69163FA8FD24CF5F" +
	"83655D23DCA3AD961C62F356208552BB9ED529077096966D" +
	"670C354E4ABC9804F1746C08CA18217C32905E462E36CE3B" +
	"E39E772C180E86039B2783A2EC07A28FB5C55DF06F4C52C9" +
	"DE2BCBF6955817183995497CEA956AE515D2261898FA0510" +
	"15728E5A8AAAC42DAD33170D04507A33A85521ABDF1CBA64" +
	"ECFB850458DBEF0A8AEA71575D060C7DB3970F85A6E1E4C7" +
	"ABF5AE8CDB0933D71E8C94E04A25619DCEE3D2261AD2EE6B" +
	"F12FFA06D98A0864D87602733EC86A64521F2B18177B200C" +
	"BBE117577A615D6C770988C0BAD946E208E24FA074E5AB31" +
	"43DB5BFCE0FD108E4B82D120A92108011A723C12A787E6D7" +
	"88719A10BDBA5B2699C327186AF4E23C1A946834B6150BDA" +
	"2583E9CA2AD44CE8DBBBC2DB04DE8EF92E8EFC141FBECAA6" +
	"287C59474E6BC05D99B2964FA090C3A2233BA186515BE7ED" +
	"1F612970CEE2D7AFB81BDD762170481CD0069127D5B05AA9" +
	"93B4EA988D8FDDC186FFB7DC90A6C08F4DF435C934063199" +
	"FFFFFFFFFFFFFFFF"

// Test that we can receive an unencrypted message on the server
func TestServerImpl_ReceiveMessageFromClient(t *testing.T) {
	// Set up the necessary globals
	g := cyclic.NewGroup(cyclic.NewIntFromString(
		PRIME, 16), cyclic.NewInt(0), cyclic.NewInt(5),
		cyclic.NewRandom(cyclic.NewInt(0), cyclic.NewInt(1000)))
	globals.Grp = &g
	MessageCh = make(chan *realtime.Slot, 1)

	// Expected values
	senderID := id.NewUserFromUint(66, t)
	recipientID := id.NewUserFromUint(65, t)
	text := []byte("hey there, sailor. want to see my unencrypted message?")

	// Create an unencrypted message for testing
	message, err := format.NewMessage(senderID, recipientID, text)
	copy(message.GetMessageInitVect(), []byte("abcdefghi"))
	copy(message.GetRecipientInitVect(), []byte("fghijklmn"))

	if err != nil {
		t.Errorf("Couldn't construct message: %s", err.Error())
	}
	messageSerial := message.SerializeMessage()

	// Send the message to the server itself
	ServerImpl{}.ReceiveMessageFromClient(&pb.CmixMessage{
		SenderID:       senderID[:],
		MessagePayload: messageSerial.MessagePayload,
		RecipientID:    messageSerial.RecipientPayload,
	})

	receivedMessage := <-MessageCh

	// Verify that the received message holds the expected data
	if *receivedMessage.CurrentID != *senderID {
		t.Errorf("Received sender ID %v, expected %v",
			*receivedMessage.CurrentID, *senderID)
	}
	result := format.DeserializeMessage(format.MessageSerial{
		MessagePayload:   receivedMessage.Message.LeftpadBytes(format.TOTAL_LEN),
		RecipientPayload: receivedMessage.EncryptedRecipient.LeftpadBytes(format.TOTAL_LEN)})
	if *result.GetSender() != *senderID {
		t.Errorf("Received sender ID in bytes %q, expected %q",
			*result.GetSender(), *senderID)
	}
	if *result.GetRecipient() != *recipientID {
		t.Errorf("Received recipient ID %q, expected %q",
			*result.GetRecipient(), *recipientID)
	}
	if !bytes.Contains(result.GetPayload(), text) {
		t.Errorf("Received payload message %q, expected %q",
			result.GetPayload(), text)
	}
}
