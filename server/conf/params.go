////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package conf

import (
	"encoding/binary"
	"github.com/spf13/viper"
	"gitlab.com/elixxir/primitives/id"
)

// This object is used by the server instance.
// A viper (or any yaml based) configuration
// can be unmarshalled into this object.
// For viper just use Unmarshal(&params).
// Note not all fields are in the YAML, ie NodeID
// but all fields must be in the viper object
type Params struct {
	Database DB
	Groups   Groups
	Paths    Paths
	Servers  []string
	Gateways []string
	NodeID   *id.Node
	SkipReg  bool `yaml:"skipReg"`
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

	nid := vip.GetUint64("nodeId")
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, nid)
	params.NodeID = id.NewNodeFromBytes(buf)

	return &params, nil
}
