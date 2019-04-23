package server

import "gitlab.com/elixxir/server/node"

type NodeAddress struct {
	cert    string
	address string
	id      uint64
}

func (na NodeAddress) DeepCopy() NodeAddress {
	return NodeAddress{na.cert, na.address, na.id}
}

type Round struct {
	id     node.RoundID
	buffer node.RoundBuffer

	nodes []NodeAddress
	myLoc int
}

func (r *Round) GetNextNodeAddress() NodeAddress {
	return r.nodes[(r.myLoc+1)%len(r.nodes)]
}

func (r *Round) GetPrevNodeAddress() NodeAddress {
	return r.nodes[(r.myLoc-1)%len(r.nodes)]
}

func (r *Round) GetNodeAddress(index int) NodeAddress {
	return r.nodes[index%len(r.nodes)]
}

func (r *Round) GetAllNodesAddress() []NodeAddress {
	nal := make([]NodeAddress, len(r.nodes))

	for i := range nal {
		nal[i] = r.nodes[i].DeepCopy()
	}
	return nal
}

func (r *Round) GetID() node.RoundID {
	return r.id
}

func (r *Round) GetBuffer() *node.RoundBuffer {
	return &r.buffer
}
