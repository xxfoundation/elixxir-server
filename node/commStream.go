////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package node

import (
	"errors"
	"gitlab.com/elixxir/comms/mixmessages"
)

var ErrOutsideOfGroup = errors.New("cyclic int is outside of the prescribed group")
var ErrOutsideOfBatch = errors.New("cyclic int is outside of the prescribed batch")

type CommsStream interface {
	Input(index uint32, slot *mixmessages.CmixSlot) error
	Output(index uint32) *mixmessages.CmixSlot
}
