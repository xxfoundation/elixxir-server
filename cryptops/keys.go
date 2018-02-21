////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package cryptops

import "gitlab.com/privategrity/crypto/cyclic"

type KeyType int

const (
	TRANSMISSION KeyType = 0
	RECEPTION    KeyType = 1
	RETURN       KeyType = 2
)

type KeySlot interface {
	//Slot of the message
	SlotID() uint64

	//ID of the user for keygen
	UserID() uint64

	//Cyclic int to place the key in
	Key() *cyclic.Int

	//Returns the KeyType
	GetKeyType() KeyType
}
