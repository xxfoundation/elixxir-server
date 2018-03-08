////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package io

import (
	pb "gitlab.com/privategrity/comms/mixmessages"
	"gitlab.com/privategrity/crypto/cyclic"
	g "gitlab.com/privategrity/server/globals"
	"testing"
)

func TestServerImpl_ClientPoll(t *testing.T) {
	user := g.Users.NewUser("test") // Create user
	g.Users.UpsertUser(user)        // Insert user into registry

	// Add a message of 42 to the user's message buffer
	user.MessageBuffer <- &pb.CmixMessage{MessagePayload: cyclic.NewInt(42).Bytes()}

	// Get the message for the valid user via ClientPoll
	msg := ServerImpl{}.ClientPoll(&pb.ClientPollMessage{UserID: user.UID})
	// Make sure the message contains the same payload
	if cyclic.NewIntFromBytes(msg.MessagePayload).Cmp(cyclic.NewInt(42)) != 0 {
		t.Errorf("ClientPoll returned invalid MessagePayload!")
	}

	// Get the empty message for the valid user via ClientPoll
	msg = ServerImpl{}.ClientPoll(&pb.ClientPollMessage{UserID: uint64(0)})
	// Make sure the message contains an empty payload because the buffer is empty
	if len(msg.MessagePayload) > 0 {
		t.Errorf("ClientPoll returned unexpected nonempty MessagePayload!")
	}

	// Get the empty message for an invalid user via ClientPoll
	msg = ServerImpl{}.ClientPoll(&pb.ClientPollMessage{UserID: uint64(5)})
	// Make sure the message contains an empty payload because the user is invalid
	if len(msg.MessagePayload) > 0 {
		t.Errorf("ClientPoll returned unexpected nonempty MessagePayload!")
	}
}
