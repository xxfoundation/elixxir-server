////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package io

import "time"

// StreamInfo is an object which tracks the start
// and end of stream reception. Used for bandwidth logging
// for streaming
type streamInfo struct {
	Start time.Time
	End   time.Time
}
