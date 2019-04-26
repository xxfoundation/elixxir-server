package services

import "gitlab.com/elixxir/primitives/id"

//Contains all information about communicating with a node
type NodeAddress struct {
	Cert    string
	Address string
	Id      id.Node
}

// DeepCopy makes a complete copy of a NodeAddress
func (na NodeAddress) DeepCopy() NodeAddress {
	return NodeAddress{na.Cert, na.Address, na.Id}
}

type NodeAddressList struct {
	nodes []NodeAddress
	myLoc int
}

// NewNodeAddressList makes a list of node addresses for use
func NewNodeAddressList(list []NodeAddress, myloc int) *NodeAddressList {
	nal := NodeAddressList{
		nodes: make([]NodeAddress, len(list)),
		myLoc: myloc,
	}

	for index, na := range list {
		nal.nodes[index] = na
	}

	return &nal
}

// GetNextNodeAddress gets the next node in the list and
func (r *NodeAddressList) GetNextNodeAddress() NodeAddress {
	return r.nodes[(r.myLoc+1)%len(r.nodes)]
}

// GetNextNodeAddress gets the pre node in the list and
func (r *NodeAddressList) GetPrevNodeAddress() NodeAddress {
	return r.nodes[(r.myLoc-1)%len(r.nodes)]
}

// GetNodeAddress Gets the node address at a specific index, wraps around if out of range
func (r *NodeAddressList) GetNodeAddress(index int) NodeAddress {
	return r.nodes[index%len(r.nodes)]
}

// GetNodeAddress Returns a copy of the internal node address list
func (r *NodeAddressList) GetAllNodesAddress() []NodeAddress {
	nal := make([]NodeAddress, len(r.nodes))

	for i := range nal {
		nal[i] = r.nodes[i].DeepCopy()
	}
	return nal
}
