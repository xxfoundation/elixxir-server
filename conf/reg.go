////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package conf

type Reg interface {
	Skip() bool
}

type regImpl struct {
	skip bool
}

func NewReg(skip bool) Reg {
	return regImpl{
		skip: skip,
	}
}

func (reg regImpl) Skip() bool {
	return reg.skip
}
