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
	Index       int // TODO: Do we need this field and how do we populate it?
	Batch       uint32
	SkipReg     bool `yaml:"skipReg"`
	Verbose     bool
	KeepBuffers bool
	Groups      Groups

	Node          Node
	Database      Database
	Gateways      Gateways
	Permissioning Permissioning
}

// NewParams gets elements of the viper object
// and updates the params object. It returns params
// unless it fails to parse in which it case returns error
func NewParams(vip *viper.Viper) (*Params, error) {

	params := Params{}

	params.Index = vip.GetInt("index")

	params.Node.Id = vip.GetString("node.id")
	params.Node.Ids = vip.GetStringSlice("node.ids")
	params.Node.Paths.Cert = vip.GetString("node.paths.cert")
	params.Node.Paths.Key = vip.GetString("node.paths.key")
	params.Node.Paths.Log = vip.GetString("node.paths.log")
	params.Node.Addresses = vip.GetStringSlice("node.addresses")

	params.Database.Name = vip.GetString("database.name")
	params.Database.Username = vip.GetString("database.username")
	params.Database.Password = vip.GetString("database.password")
	params.Database.Addresses = vip.GetStringSlice("database.addresses")

	params.Gateways.Paths.Cert = vip.GetString("gateways.paths.cert")
	params.Gateways.Addresses = vip.GetStringSlice("gateways.addresses")

	params.Permissioning.Paths.Cert = vip.GetString("permissioning.paths.cert")
	params.Permissioning.Address = vip.GetString("permissioning.address")
	params.Permissioning.RegistrationCode = vip.GetString("permissioning.registrationCode")

	params.Batch = vip.GetUint32("batch")
	params.SkipReg = vip.GetBool("skipReg")
	params.Verbose = vip.GetBool("verbose")
	params.KeepBuffers = vip.GetBool("keepBuffers")

	params.Groups.CMix = vip.GetStringMapString("groups.cmix")
	params.Groups.E2E = vip.GetStringMapString("groups.e2e")

	return &params, nil
}
