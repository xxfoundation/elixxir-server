////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package conf

type Permissioning struct {
	Paths   Paths
	Address string
	RegCode string `yaml:"regCode"`
}
