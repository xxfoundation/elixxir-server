////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package conf

import "github.com/pkg/errors"

type Registration struct {
	SkipReg bool
	enable bool
}

func (reg *Registration) SetReg(skip bool) error {

	if !reg.enable {
		return errors.Errorf("SetDB cannot be called since DB wasn't init. correctly")
	}

	reg.SkipReg = skip

	return nil
}