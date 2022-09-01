////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package conf

// Contains graph generator config params
type GraphGen struct {
	minInputSize    uint32
	defaultNumTh    uint8
	outputSize      uint32
	outputThreshold float32
}
