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

func (reg *Registration) SetRegistry(skip bool) error {

	if !reg.enable {
		return errors.Errorf("SetRegistry failed due to improper init.")
	}

	reg.SkipReg = skip
	reg.enable = false

	return nil
}