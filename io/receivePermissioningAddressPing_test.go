////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package io

import (
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/server/internal"
	"gitlab.com/elixxir/server/internal/state"
	"gitlab.com/elixxir/server/testUtil"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/primitives/id"
	"testing"
)

// Happy path.
func TestReceivePermissioningAddressPing(t *testing.T) {
	def := internal.Definition{
		RngStreamGen: fastRNG.NewStreamGenerator(8, 8, csprng.NewSystemRNG),
		FullNDF:      testUtil.NDF,
		PartialNDF:   testUtil.NDF,
		Flags:        internal.Flags{},
		Gateway:      internal.GW{ID: &id.TempGateway},
		Network:      internal.Perm{Address: "0.0.0.0:10"},
		DevMode:      true,
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
