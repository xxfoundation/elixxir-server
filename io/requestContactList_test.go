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
)

func TestRequestContactList(t *testing.T) {
	user := globals.Users.NewUser("test") // Create user
	user.Nick = "Michael"
	globals.Users.UpsertUser(user) // Insert user into registry
	user = globals.Users.NewUser("test")
	user.Nick = "Me"
	globals.Users.UpsertUser(user)

	// Currently we just return all the nicks
	contacts, err := mixclient.RequestContactList(NextServer, &pb.ContactPoll{})
	if err != nil {
		t.Errorf("RequestContactList() returned an error: %v", err.Error())
	}

	expectedNicks := []string{"Michael", "Me"}
	nicksFound := make([]bool, len(expectedNicks))

	for i := 0; i < len(expectedNicks); i++ {
		// look for the expected nick somewhere in the returned map
		for j := 0; j < len(contacts.Contacts); j++ {
			if contacts.Contacts[i] == nil {
				t.Errorf("Got a nil nick at index %v.", i)
			}
			if contacts.Contacts[j].Nick == expectedNicks[i] {
				nicksFound[i] = true
				break
			}
		}
		if !nicksFound[i] {
			t.Errorf("Couldn't find the expected nick at index %v. "+
				"expected: %v", i, expectedNicks[i])
		}
	}
}
