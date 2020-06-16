///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package services

import "fmt"

func NameStringer(ipAddr string, loc, numNodes int) string {
	return fmt.Sprintf("%s - (%d/%d)", ipAddr, loc+1, numNodes)
}
