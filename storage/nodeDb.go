///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

// Handles the database ORM for nodes

package storage

import (
	"context"
	"errors"
	jww "github.com/spf13/jwalterweatherman"
)

// Helper for forcing panics in the event of a CDE, otherwise acts as a pass-through
func catchCde(err error) error {
	if errors.Is(err, context.DeadlineExceeded) {
		jww.FATAL.Panicf("Database call timed out: %+v", err.Error())
	}
	return err
}
