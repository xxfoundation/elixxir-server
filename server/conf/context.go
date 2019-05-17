////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package conf

// Should this be called localContext or View/LocalView?
type Context interface {
	GetServers() []string
	ID() uint64
	// batch size?
}

type contextImpl struct {
	servers []string
	id      uint64
}

func NewContext(servers []string, nodeId uint64) Context {
	return contextImpl{
		servers: servers,
		id:      nodeId,
	}
}

func (node contextImpl) GetServers() []string {
	return node.servers
}

func (node contextImpl) ID() uint64 {
	return node.id
}
