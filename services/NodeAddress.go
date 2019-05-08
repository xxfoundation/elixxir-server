package services

import "gitlab.com/elixxir/primitives/id"

type NodeIDList struct {
	nodes []*id.Node
	myLoc int
}

// TODO make sure all these node Ids are properly connected to through the
//  network definition file
// NewNodeIDList makes a list of node IDs that can be used to identify
// connections for comms
func NewNodeIDList(list []*id.Node, myloc int) *NodeIDList {
	nal := NodeIDList{
		nodes: make([]*id.Node, len(list)),
		myLoc: myloc,
	}

	for index, nid := range list {
		nal.nodes[index] = nid
	}

	return &nal
}

// GetNextNodeAddress gets the next node in the list and
func (nar *NodeIDList) GetNextNodeID() *id.Node {
	return nar.nodes[(nar.myLoc+1)%len(nar.nodes)]
}

// GetNextNodeAddress gets the pre node in the list and
func (nar *NodeIDList) GetPrevNodeID() *id.Node {
	return nar.nodes[(nar.myLoc-1)%len(nar.nodes)]
}

// GetNodeAddress Gets the node address at a specific index, wraps around if out of range
func (nar *NodeIDList) GetNodeID(index int) *id.Node {
	return nar.nodes[index%len(nar.nodes)]
}

// GetNodeAddress Returns a copy of the internal node address list
func (nar *NodeIDList) GetAllNodeIDs() []*id.Node {
	nal := make([]*id.Node, len(nar.nodes))

	for i := range nal {
		nal[i] = nar.nodes[i].DeepCopy()
	}
	return nal
}

//Returns true if the node is the first node
func (nar *NodeIDList) IsFirstNode() bool {
	return nar.myLoc == 0
}

//Returns true if the node is the last node
func (nar *NodeIDList) IsLastNode() bool {
	return nar.myLoc == len(nar.nodes)-1
}
