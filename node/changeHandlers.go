////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package node

import (
	"github.com/pkg/errors"
	nodeComms "gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/permissioning"
	"gitlab.com/elixxir/server/server/state"
)

// todo connect o perm here
func NotStarted(from current.Activity) error {
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
	newNdf, err := permissioning.PollNdf(def, network, gatewayNdfChan, gatewayReadyCh, permHost)
	if err != nil {
		return errors.Errorf("Failed to get ndf: %+v", err)
	}

	network.Shutdown()



	return nil
}

func Waiting(from current.Activity) error {
	// start waiting process
	return nil
}

func Precomputing(from current.Activity) error {
	// start pre-precomputation

	return nil
}

func Standby(from current.Activity) error {
	// start standby process
	return nil

}

func Realtime(from current.Activity) error {
	// start realtime
	return nil

}

func Completed(from current.Activity) error {
	// start completed
	return nil
}

func NewStateChanges() [current.NUM_STATES]state.Change {
	//return state changes arr
	//create the state change function table
	var stateChanges [current.NUM_STATES]state.Change

	stateChanges[current.NOT_STARTED] = NotStarted
	stateChanges[current.WAITING] = Waiting
	stateChanges[current.PRECOMPUTING] = Precomputing
	stateChanges[current.STANDBY] = Standby
	stateChanges[current.REALTIME] = Realtime
	stateChanges[current.COMPLETED] = Completed

	return stateChanges
}
