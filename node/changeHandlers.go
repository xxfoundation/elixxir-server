////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package node

// fixme: add file description

import (
	"github.com/pkg/errors"
	"github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/comms/connect"
	nodeComms "gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/permissioning"
	"gitlab.com/elixxir/server/server"
	"gitlab.com/elixxir/server/server/round"
	"gitlab.com/elixxir/server/server/state"
)

func Dummy(from current.Activity) error {
	return nil
}

// Beginning state of state machine, enters waiting upon successful completion
func NotStarted(def *server.Definition, instance *server.Instance) error {
	// all the server startup code
	impl := nodeComms.NewImplementation()

	// instance.get
	// Start comms network
	network := nodeComms.StartNode(def.ID.String(), def.Address, impl, def.TlsCert, def.TlsKey)
	_, err := network.AddHost(id.NewTmpGateway().String(), def.Gateway.Address, def.Gateway.TlsCert, true, true)
	if err != nil {
		return errors.Errorf("Unable to add gateway host: %+v", err)
	}
	// Connect to the Permissioning Server without authentication
	permHost, err := network.AddHost(id.PERMISSIONING,
		// instance.GetPermissioningAddress,
		def.Permissioning.Address, def.Permissioning.TlsCert, true, false)
	if err != nil {
		return errors.Errorf("Unable to connect to registration server: %+v", err)
	}

	// Blocking call: Begin Node registration
	err = permissioning.RegisterNode(def, network, permHost)
	if err != nil {
		return errors.Errorf("Failed to register node: %+v", err)
	}

	// Disconnect the old permissioning server to enable authentication
	permHost.Disconnect()

	// Connect to the Permissioning Server with authentication enabled
	permHost, err = network.AddHost(id.PERMISSIONING,
		def.Permissioning.Address, def.Permissioning.TlsCert, true, true)
	if err != nil {
		return errors.Errorf("Unable to connect to registration server: %+v", err)
	}

	// Blocking call: Request ndf from permissioning
	err = permissioning.PollNdf(def, network, permHost, instance)
	if err != nil {
		return errors.Errorf("Failed to get ndf: %+v", err)
	}

	network.Shutdown()

	// Parse the Ndf
	serverCert, gwCert, err := permissioning.InstallNdf(def, instance.GetConsensus().GetFullNdf().Get())
	if err != nil {
		return errors.Errorf("Failed to install ndf: %+v", err)
	}
	def.TlsCert = []byte(serverCert)
	def.Gateway.TlsCert = []byte(gwCert)

	return nil
}

// fixme: doc string
func Waiting(from current.Activity) error {
	// start waiting process
	return nil
}

// fixme: doc string
func Precomputing(instance *server.Instance, newRoundTimeout int) state.Change {
	// Add round.queue to instance, get that here and use it to get new round
	// start pre-precomputation
	roundInfo := <-instance.GetCreateRoundQueue()
	roundID := roundInfo.GetRoundId()
	topology := roundInfo.GetTopology()

	// Extract topology from RoundInfo
	// TODO: create NewTopology function which accepts list of strings
	nodeIDs := make([]*id.Node, 0)
	for _, s := range topology {
		nodeIDs = append(nodeIDs, id.NewNodeFromBytes([]byte(s)))
	}
	circuit := connect.NewCircuit(nodeIDs)

	//Build the components of the round
	phases, phaseResponses := NewRoundComponents(
		instance.GetGraphGenerator(),
		circuit,
		instance.GetID(),
		instance,
		instance.GetBatchSize(),
		newRoundTimeout)

	//Build the round
	rnd := round.New(
		instance.GetGroup(),
		instance.GetUserRegistry(),
		roundID, phases, phaseResponses,
		circuit,
		instance.GetID(),
		instance.GetBatchSize(),
		instance.GetRngStreamGen(),
		instance.GetIP())

	//Add the round to the manager
	instance.GetRoundManager().AddRound(rnd)

	jwalterweatherman.INFO.Printf("[%s]: RID %d CreateNewRound COMPLETE", instance,
		roundID)

	if circuit.IsFirstNode(instance.GetID()) {
		err := StartLocalPrecomp(instance, roundID)
		if err != nil {
			return nil, errors.WithMessage(err, "Failed to TransmitCreateNewRound")
		}
	}

	return nil, nil
}

// fixme: doc string
func Standby(from current.Activity) error {
	// start standby process
	return nil

}

// fixme: doc string
func Realtime(from current.Activity) error {
	// start realtime
	return nil

}

// fixme: doc string
func Completed(from current.Activity) error {
	// start completed
	return nil
}

// fixme: doc string
func NewStateChanges() [current.NUM_STATES]state.Change {
	//return state changes arr
	//create the state change function table
	var stateChanges [current.NUM_STATES]state.Change

	stateChanges[current.NOT_STARTED] = Dummy
	stateChanges[current.WAITING] = Dummy
	stateChanges[current.PRECOMPUTING] = Dummy
	stateChanges[current.STANDBY] = Dummy
	stateChanges[current.REALTIME] = Dummy
	stateChanges[current.COMPLETED] = Dummy

	return stateChanges
}
