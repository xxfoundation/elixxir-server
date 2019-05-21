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

	if !context.enable {
		return errors.Errorf("SetContext failed due to improper init.")
	}

	context.Servers = servers
	context.NodeID = nodeId
	context.enable = false

	return nil
}
