package round

import (
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/server/phase"
)

type Round struct {
	id     id.Round
	buffer *Buffer

	nodeAddressList *NodeAddressList
	state           *phase.StateGroup

	//on first node and last node the phases vary
	phaseMap map[phase.Type]int
	phases   []*phase.Phase
}

// Creates and initializes a new round, including all phases
func New(grp *cyclic.Group, id id.Round, phases []*phase.Phase, nodes []NodeAddress, myLoc int, batchSize uint32) *Round {

	round := Round{}
	round.id = id

	maxBatchSize := uint32(0)

	stateGroup := phase.NewStateGroup()
	round.state = stateGroup

	for _, p := range phases {
		if p.GetGraph().GetExpandedBatchSize() > maxBatchSize {
			maxBatchSize = p.GetGraph().GetExpandedBatchSize()
		}
		p.ConnectToRound(id, stateGroup)
	}

	round.buffer = NewBuffer(grp, batchSize, maxBatchSize)

	for index, p := range phases {
		p.GetGraph().Link(&round)
		round.phaseMap[p.GetType()] = index
	}

	round.phases = make([]*phase.Phase, len(phases))

	copy(round.phases[:], phases[:])

	round.nodeAddressList = NewNodeAddressList(nodes, myLoc)

	return &round
}

func (r *Round) GetID() id.Round {
	return r.id
}

func (r *Round) GetBuffer() *Buffer {
	return r.buffer
}

func (r *Round) GetPhase(p phase.Type) *phase.Phase {
	if int(p) > len(r.phases) {
		return nil
	}
	return r.phases[r.phaseMap[p]]
}

func (r *Round) GetCurrentPhase() phase.Type {
	return r.state.GetCurrentPhase()
}

func (r *Round) GetNodeAddressList() *NodeAddressList {
	return r.GetNodeAddressList()
}
