////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package io

import (
	pb "gitlab.com/privategrity/comms/mixmessages"
	"gitlab.com/privategrity/server/globals"
)

func (s ServerImpl) ClientGetContactList(inputMsg *pb.
	ContactPoll) *pb.ContactMessage {
	userCount := globals.Users.CountUsers()

	contactList := pb.ContactMessage{
		Contacts: make([]*pb.Contact, userCount),
	}

	idList, nickList := globals.Users.GetNickList()

	for i := 0; i < userCount; i++ {
		contactList.Contacts[i].Nick = nickList[i]
		contactList.Contacts[i].UserID = idList[i]
	}

	return &contactList
}
