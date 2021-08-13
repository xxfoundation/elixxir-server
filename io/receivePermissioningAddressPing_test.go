////////////////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                                       //
//                                                                                        //
// Use of this source code is governed by a license that can be found in the LICENSE file //
////////////////////////////////////////////////////////////////////////////////////////////

package io

import (
	"git.xx.network/elixxir/primitives/current"
	"git.xx.network/elixxir/server/internal"
	"git.xx.network/elixxir/server/internal/state"
	"git.xx.network/elixxir/server/testUtil"
	"git.xx.network/xx_network/primitives/id"
	"testing"
)

// Happy path.
func TestReceivePermissioningAddressPing(t *testing.T) {
	def := internal.Definition{
		FullNDF:    testUtil.NDF,
		PartialNDF: testUtil.NDF,
		Flags:      internal.Flags{},
		Gateway:    internal.GW{ID: &id.TempGateway},
		Network:    internal.Perm{Address: "0.0.0.0:10"},
		DevMode:    true,
	}
	m := state.NewTestMachine(dummyStates, current.ERROR, t)
	instance, _ := internal.CreateServerInstance(&def, NewImplementation, m, "")

	addr, err := ReceivePermissioningAddressPing(instance)
	if err != nil {
		t.Errorf("ReceivePermissioningAddressPing returned an erro: %+v", err)
	}
	if def.Network.Address != addr {
		t.Errorf("ReceivePermissioningAddressPing() failed to return the "+
			"expected address for permissioning."+
			"\n\texpected: %s\n\treceived: %s", def.Network.Address, addr)
	}
	t.Log(addr)
}
