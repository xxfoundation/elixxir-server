////////////////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                                       //
//                                                                                        //
// Use of this source code is governed by a license that can be found in the LICENSE file //
////////////////////////////////////////////////////////////////////////////////////////////

package io

import (
	"gitlab.com/elixxir/server/internal"
)

// ReceivePermissioningAddressPing returns the permissioning server's address.
func ReceivePermissioningAddressPing(instance *internal.Instance) (string, error) {
	return instance.GetDefinition().Network.Address, nil
}
