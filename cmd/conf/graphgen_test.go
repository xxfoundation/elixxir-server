///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package conf

import "runtime"

var ExpectedGraphGen = GraphGen{
	minInputSize:    4,
	defaultNumTh:    uint8(runtime.NumCPU()),
	outputSize:      4,
	outputThreshold: 0.0,
}
