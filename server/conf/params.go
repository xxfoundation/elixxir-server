////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package conf

import (
	"github.com/spf13/viper"
)

// This object is used by the server instance.
// A viper (or any yaml based) configuration
// can be unmarshalled into this object.
// For viper just use Unmarshal(&params).
// Note not all fields are in the YAML, ie NodeID
// but all fields must be in the viper object
type Params struct {
	//Node Identity Params
	Index    int
	Database DB
	SkipReg  bool `yaml:"skipReg"`

	//Network Identity Params
	Groups        Groups
	Paths         Paths
	NodeAddresses []string
	// these are base64 strings, so instance creation must base64 decode these
	// before using them as node IDs
	NodeIDs  []string
	Gateways []string
	Batch    uint32
}

// NewParams returns a params object if it is able to
// unmarshal the viper config, otherwise it returns
// an error.
func NewParams(vip *viper.Viper) (*Params, error) {

	params := Params{}
	err := vip.Unmarshal(&params)
	if err != nil {
		return nil, err
	}

	params.Groups = NewGroups(vip)

	return &params, nil
}
