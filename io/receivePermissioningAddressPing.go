package io

import (
	"gitlab.com/elixxir/server/internal"
)

// ReceivePermissioningAddressPing returns the permissioning server's address.
func ReceivePermissioningAddressPing(instance *internal.Instance) (string, error) {
	return instance.GetDefinition().Permissioning.Address, nil
}
