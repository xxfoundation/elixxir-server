////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package services

// Struct which contains a chunk of cryptographic data to be operated on
type Slot interface {
	//Slot of the message
	SlotID() uint64
}
