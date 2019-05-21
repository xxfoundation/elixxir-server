////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package conf

import "github.com/pkg/errors"

type Context struct {
	Servers []string
	NodeID  uint64
	enable  bool
}

func (context *Context) SetContext(servers []string, nodeId uint64) error {

	// Check if setting values is enabled
	if !context.enable {
		return errors.Errorf("SetContext failed due to improper init.")
	}

	// Check if input fields are valid
	// ...


	// Set the values
	context.Servers = servers
	context.NodeID = nodeId

	// Disable updating values
	context.enable = false

	return nil
}
