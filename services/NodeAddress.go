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
func (nar *NodeAddressList) GetNextNodeAddress() NodeAddress {
	return nar.nodes[(nar.myLoc+1)%len(nar.nodes)]
}

// GetNextNodeAddress gets the pre node in the list and
func (nar *NodeAddressList) GetPrevNodeAddress() NodeAddress {
	return nar.nodes[(nar.myLoc-1)%len(nar.nodes)]
}

// GetNodeAddress Gets the node address at a specific index, wraps around if out of range
func (nar *NodeAddressList) GetNodeAddress(index int) NodeAddress {
	return nar.nodes[index%len(nar.nodes)]
}

// GetNodeAddress Returns a copy of the internal node address list
func (nar *NodeAddressList) GetAllNodesAddress() []NodeAddress {
	nal := make([]NodeAddress, len(nar.nodes))

	for i := range nal {
		nal[i] = nar.nodes[i].DeepCopy()
	}
	return nal
}

//Returns true if the node is the first node
func (nar *NodeAddressList) IsFirstNode() bool {
	return nar.myLoc == 0
}

//Returns true if the node is the last node
func (nar *NodeAddressList) IsLastNode() bool {
	return nar.myLoc == len(nar.nodes)-1
}
