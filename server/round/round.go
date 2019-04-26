package round

import (
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/server/phase"
	"gitlab.com/elixxir/server/services"
)

type Round struct {
	id     id.Round
	buffer *Buffer

	nodeAddressList *services.NodeAddressList
	state           *phase.StateGroup

	//on first node and last node the phases vary
	phaseMap map[phase.Type]int
	phases   []*phase.Phase
}

// Creates and initializes a new round, including all phases
func New(grp *cyclic.Group, id id.Round, phases []*phase.Phase, nodes []services.NodeAddress, myLoc int, batchSize uint32) *Round {

	round := Round{}
	round.id = id

	maxBatchSize := uint32(0)

	stateGroup := phase.NewStateGroup()
	round.state = stateGroup

	for _, p := range phases {
		p.GetGraph().Build(batchSize)
		if p.GetGraph().GetExpandedBatchSize() > maxBatchSize {
			maxBatchSize = p.GetGraph().GetExpandedBatchSize()
		}
		p.ConnectToRound(id, stateGroup)
	}

	round.buffer = NewBuffer(grp, batchSize, maxBatchSize)
	if round.phaseMap == nil {
		round.phaseMap = make(map[phase.Type]int)
	}

	for index, p := range phases {
		p.GetGraph().Link(grp, &round)
		round.phaseMap[p.GetType()] = index
	}

	round.phases = make([]*phase.Phase, len(phases))

	copy(round.phases[:], phases[:])

	round.nodeAddressList = services.NewNodeAddressList(nodes, myLoc)

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

func (r *Round) GetNodeAddressList() *services.NodeAddressList {
	return r.nodeAddressList
}
