////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package conf

// Contains DB config params
type DB struct {
	Name      string
	Username  string
	Password  string
	Addresses []string
}
