////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package io

import (
	"gitlab.com/privategrity/comms/client"
	pb "gitlab.com/privategrity/comms/mixmessages"
	"gitlab.com/privategrity/server/globals"
	"testing"
)

func TestSetNick(t *testing.T) {
	var user *globals.User
	user = globals.Users.NewUser("Test")
	user.Nick = "Nick"
	globals.Users.UpsertUser(user)

	_, err := client.SetNick(NextServer, &pb.Contact{
		UserID: user.ID,
		Nick:   "Jake",
	})
	if err != nil {
		t.Errorf("SetNick() returned an error: %v", err.Error())
	}

	expectedNick := "Jake"
	user, err = globals.Users.GetUser(user.ID)
	if err != nil {
		t.Errorf("User with id %v mysteriously disappeared from the user"+
			" registry", user.ID)
	}
	if user.Nick != expectedNick {
		t.Errorf("Nick differed from expected. Got: %v, expected %v",
			user.Nick, expectedNick)
	}
}
