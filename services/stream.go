////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package services

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/cyclic"
)

var ErrOutsideOfGroup = errors.New("cyclic int is outside of the prescribed group")
var ErrOutsideOfBatch = errors.New("cyclic int is outside of the prescribed batch")

// Stream defines the interface that Graphs must adhere to in order to be interfaced with,
// which functionally describes how IO and local data goes in and out of self-contained Graphs
type Stream interface {
	GetName() string
	Link(grp *cyclic.Group, BatchSize uint32, source ...interface{})
	Input(index uint32, slot *mixmessages.Slot) error
	Output(index uint32) *mixmessages.Slot
}
