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
// It should be constructed using a viper object
type Params struct {
	Index         int // TODO: Do we need this field and how do we populate it?
	Node          Node
	Database      Database
	Gateways      Gateways
	Permissioning Permissioning
	Global        Global
}

// NewParams unmarshals a viper object and returns
// the params object unless it fails to parse
func NewParams(vip *viper.Viper) (*Params, error) {

	params := Params{}
	err := vip.Unmarshal(&params)
	if err != nil {
		return nil, err
	}

	return &params, nil

}
