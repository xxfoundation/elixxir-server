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

func (reg *Registration) SetRegistration(skipReg bool) error {

	if !reg.enable {
		return errors.Errorf("SetRegistration failed due to improper init.")
	}

	reg.SkipReg = skipReg

	reg.enable = false

	return nil
}