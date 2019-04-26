package server

import (
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/server/globals"
	"gitlab.com/elixxir/server/server/phase"
	"gitlab.com/elixxir/server/server/round"
)

// Holds long-lived server state
type Instance struct {
	roundManager  *round.Manager
	resourceQueue *ResourceQueue
	grp           *cyclic.Group
	userReg       globals.UserRegistry
}

func (i *Instance) GetGroup() *cyclic.Group {
	return i.grp
}

func (i *Instance) GetUserRegistry() globals.UserRegistry {
	return i.userReg
}

func (i *Instance) GetRoundManager() *round.Manager {
	return i.roundManager
}

func (i *Instance) GetResourceQueue() *ResourceQueue {
	return i.resourceQueue
}

// Create a server instance. To actually kick off the server,
// call Run() on the resulting ServerIsntance.
func CreateServerInstance(grp *cyclic.Group, db globals.UserRegistry) *Instance {
	instance := Instance{
		roundManager: round.NewManager(),
		grp:          grp,
	}
	instance.resourceQueue = &ResourceQueue{
		// these are the phases
		phaseQueue: make(chan *phase.Phase, 5000),
		// there will only active phase, and this channel is used to kill it
		finishChan: make(chan *phase.Phase, 1),
	}
	instance.userReg = db
	return &instance
}

func (i *Instance) Run() {
	go queueRunner(i)
}
