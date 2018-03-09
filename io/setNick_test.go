////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package io

import (
	"testing"
	"gitlab.com/privategrity/server/globals"
	"gitlab.com/privategrity/comms/mixclient"
	pb "gitlab.com/privategrity/comms/mixmessages"
)

func TestSetNick(t *testing.T) {
	var user *globals.User
	user = globals.Users.NewUser("Test")
	user.Nick = "Nick"
	globals.Users.UpsertUser(user)

	_, err := mixclient.SetNick(NextServer, &pb.Contact{
		UserID: user.UID,
		Nick:   "Jake",
	})
	if err != nil {
		t.Errorf("SetNick() returned an error: %v", err.Error())
	}

	expectedNick := "Jake"
	user, ok := globals.Users.GetUser(user.UID)
	if !ok {
		t.Errorf("User with id %v mysteriously disappeared from the user" +
			" registry", user.UID)
	}
	if user.Nick != expectedNick {
		t.Errorf("Nick differed from expected. Got: %v, expected %v",
			user.Nick, expectedNick)
	}
}
