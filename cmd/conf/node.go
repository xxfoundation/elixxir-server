////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package conf

// Contains Node config params
type Node struct {
	Id        string
	Ids       []string
	Paths     Paths
	Addresses []string
}
