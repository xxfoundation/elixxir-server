////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package services

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/cyclic"
)

var ErrOutsideOfGroup = errors.New("cyclic int is outside of the prescribed group")
var ErrOutsideOfBatch = errors.New("cyclic int is outside of the prescribed batch")

type Stream interface {
	GetName() string
	Link(grp *cyclic.Group, BatchSize uint32, source ...interface{})
	Input(index uint32, slot *mixmessages.Slot) error
	Output(index uint32) *mixmessages.Slot
}
