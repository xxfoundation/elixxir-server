////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package conf

// A viper configuration can be unmarshalled
// into this object using Unmarshal(&params)
type Params struct {
	database     DB
	groups       Groups
	paths        Paths
	servers 	 []string
	nodeID  	 uint64
	skipReg 	 bool
}
