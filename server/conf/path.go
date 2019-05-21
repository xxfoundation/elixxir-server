////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package conf

import "github.com/pkg/errors"

type Paths struct {
	CertPath string
	KeyPath  string
	LogPath  string
	enable   bool
}

func (paths *Paths) SetPaths(cert, key, log string) error {

	if !paths.enable {
		return errors.Errorf("SetDB cannot be called since DB wasn't init. correctly")
	}

	paths.CertPath = cert
	paths.KeyPath = key
	paths.LogPath = log

	return nil
}
