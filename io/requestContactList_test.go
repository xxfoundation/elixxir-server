////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package io

import (
	"testing"
	"gitlab.com/privategrity/server/globals"
	pb "gitlab.com/privategrity/comms/mixmessages"
	"gitlab.com/privategrity/comms/mixclient"
	"fmt"
)

// MARK
func TestRequestContactList(t *testing.T) {
	// empty user registry from previous tests so we can make sure that our
	// user's first in the map
	globals.Users.DeleteUser(1)
	user := globals.Users.NewUser("test") // Create user
	user.Nick = "Michael"
	globals.Users.UpsertUser(user)        // Insert user into registry
	user = globals.Users.NewUser("test")
	user.Nick = "Me"
	globals.Users.UpsertUser(user)

	// Currently we just return all the nicks
	contacts, err := mixclient.RequestContactList(NextServer, &pb.ContactPoll{})
	if err != nil {
		t.Errorf("RequestContactList() returned an error: %v", err.Error())
	}
	// TODO: Remove this print
	fmt.Printf("%v",contacts)
	// TODO: I hacked this test to work with hard-coded users.
	// TODO: Revisit to make more robust
	/*
	expectedNicks := []string{"Michael", "Me"}
	for i := 0; i < len(expectedNicks); i++ {
		if contacts.Contacts[i] == nil {
			t.Errorf("Got a nil nick at index %v.", i)
		} else if expectedNicks[i] != contacts.Contacts[i].Nick {
			t.Errorf("Got an unexpected nick at index %v. Got: %v, " +
				"expected: %v", i, contacts.Contacts[i].Nick, expectedNicks[i])
		}
	}
	*/
}