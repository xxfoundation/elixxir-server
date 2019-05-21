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

	// Check if SetPaths is enabled
	if !paths.enable {
		return errors.Errorf("SetPaths failed due to improper init.")
	}

	// Check if input fields are valid
	// ...


	// Set the values
	paths.CertPath = cert
	paths.KeyPath = key
	paths.LogPath = log

	// Disable updating values
	paths.enable = false

	return nil
}
