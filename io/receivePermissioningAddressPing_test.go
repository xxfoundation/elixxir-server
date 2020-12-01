package io

import (
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/server/internal"
	"gitlab.com/elixxir/server/internal/state"
	"gitlab.com/elixxir/server/testUtil"
	"gitlab.com/xx_network/primitives/id"
	"testing"
)

// Happy path.
func TestReceivePermissioningAddressPing(t *testing.T) {
	def := internal.Definition{
		FullNDF:       testUtil.NDF,
		PartialNDF:    testUtil.NDF,
		Flags:         internal.Flags{DisableIpOverride: true},
		Gateway:       internal.GW{ID: &id.TempGateway},
		Permissioning: internal.Perm{Address: "0.0.0.0:10"},
	}
	m := state.NewTestMachine(dummyStates, current.ERROR, t)
	instance, _ := internal.CreateServerInstance(&def, NewImplementation, m, "")

	addr, err := ReceivePermissioningAddressPing(instance)
	if err != nil {
		t.Errorf("ReceivePermissioningAddressPing returned an erro: %+v", err)
	}
	if def.Permissioning.Address != addr {
		t.Errorf("ReceivePermissioningAddressPing() failed to return the "+
			"expected address for permissioning."+
			"\n\texpected: %s\n\treceived: %s", def.Permissioning.Address, addr)
	}
	t.Log(addr)
}
