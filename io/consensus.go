///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////
package io

import (
	"gitlab.com/elixxir/server/internal"
)

// consensus.go contains handlers and senders for communication with
// our consensus platform

func GetNdf(instance *internal.Instance) ([]byte, error) {
	return instance.GetConsensus().GetFullNdf().Get().Marshal()
}
