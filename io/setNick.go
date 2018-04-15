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

func (s ServerImpl) SetNick(inputMsg *pb.Contact) {
	// TODO: users should only be able to set their own nicks,
	// and should get errors back otherwise
	user, err := globals.Users.GetUser(inputMsg.UserID)

	if err == nil {
		user.Nick = inputMsg.Nick
		globals.Users.UpsertUser(user)
	}
}
