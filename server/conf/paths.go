////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package conf

// Paths contains the config params for
// required file paths used by the system
// TODO: maybe create a paths object
// and have this one contain the actual file obj
type Paths struct {
	Cert string
	Key  string
	Log  string
}
