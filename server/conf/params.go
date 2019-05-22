////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package conf

// This object is used by the server instance.
// A viper (or any yaml based) configuration
// can be unmarshalled into this object.
// For viper just use Unmarshal(&params).
type Params struct {
	Database DB
	Groups   Groups
	Paths    Paths
	Servers  []string
	NodeID   int		`yaml:"nodeId"`
	SkipReg  bool 		`yaml:"skipReg"`
}
