///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package node

import (
	"github.com/pkg/errors"
	"git.xx.network/elixxir/server/internal"
	"git.xx.network/xx_network/primitives/id"
)

func GetDefaultPanicHandler(i *internal.Instance, roundID id.Round) func(g, m string, err error) {
	return func(g, m string, err error) {

		roundErr := errors.Errorf("Error in module %s of graph %s: %+v", g,
			m, err)
		i.ReportRoundFailure(roundErr, i.GetID(), roundID)
	}
}
