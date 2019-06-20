////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package conf

type Global struct {
	Batch   uint32
	SkipReg bool `yaml:"skipReg"`
	Groups  Groups
}
