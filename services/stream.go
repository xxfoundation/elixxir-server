////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package services

type Stream interface {
	GetStreamName() string
	Link(BatchSize uint32, source ...interface{})
}
