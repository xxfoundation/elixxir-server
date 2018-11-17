////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package io

import (
	jww "github.com/spf13/jwalterweatherman"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/crypto/id"
	"gitlab.com/elixxir/server/globals"
	"time"
)

// Broadcast a UserUpsert message to all servers
func UserUpsertBroadcast(userId *id.UserID, userPublicKey []byte) {
	for i := 0; i < len(Servers); {
		msg := pb.UpsertUserMessage{
			NodeID:        globals.GetNodeID(),
			UserID:        userId[:],
			UserPublicKey: userPublicKey,
			Nonce:         make([]byte, 0),
			DsaSignature:  make([]byte, 0),
		}
		_, err := node.SendUserUpsert(Servers[i], &msg)
		jww.INFO.Printf("Sending Upsert User %q to %s", userId, Servers[i])
		if err != nil {
			time.Sleep(250 * time.Millisecond)
		} else {
			i++
		}
	}
}

// Handle reception of an UpsertUser Message
// Stores UserPublicKey in the User backend
func (s ServerImpl) UserUpsert(message *pb.UpsertUserMessage) {

}
