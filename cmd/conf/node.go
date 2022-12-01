////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package conf

// Contains Node config params
type Node struct {
	Paths            Paths
	Port             int
	PublicAddress    string // Server's public address (with port)
	ListeningAddress string // Server's internal address (with port)
	InterconnectPort int
}
