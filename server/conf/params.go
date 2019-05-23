////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package conf

import "gitlab.com/elixxir/crypto/cyclic"

// This object is used by the server instance.
type Params struct {
	DBName      string
	DBUsername  string
	DBPassword  string
	DBAddresses []string

	CMix *cyclic.Group
	E2E  *cyclic.Group

	CertPath string
	KeyPath  string
	LogPath  string

	Servers  []string
	Gateways []string

	NodeID    uint64
	SkipReg   bool
	BatchSize uint64
	ServerIdx int
}
