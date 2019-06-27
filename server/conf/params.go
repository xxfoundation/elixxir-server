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
type Params struct {
	//Node Identity Params
	Index       int
	Database    DB
	SkipReg     bool
	KeepBuffers bool
	Path        Paths

	//Network Identity Params
	Batch         uint32
	Groups        Groups
	RegServerPK   string
	NodeAddresses []string
	// these are base64 strings, so instance creation must base64 decode these
	// before using them as node IDs
	NodeIDs  []string
	Gateways []string
}

// NewParams returns a params object if it is able to
// unmarshal the viper config, otherwise it returns
// an error.
func NewParams(vip *viper.Viper) (*Params, error) {

	params := Params{}

	params.Index = vip.GetInt("index")

	params.Database.Name = vip.GetString("database.name")
	params.Database.Username = vip.GetString("database.username")
	params.Database.Password = vip.GetString("database.password")
	params.Database.Addresses = vip.GetStringSlice("database.addresses")

	params.SkipReg = vip.GetBool("skipReg")
	params.KeepBuffers = vip.GetBool("keepBuffers")

	params.Path.Cert = vip.GetString("path.cert")
	params.Path.GatewayCert = vip.GetString("path.gateway_cert")
	params.Path.Key = vip.GetString("path.key")
	params.Path.Log = vip.GetString("path.log")

	params.Batch = vip.GetUint32("batch")
	params.Groups.CMix = toGroup(vip.GetStringMapString("groups.cmix"))
	params.Groups.E2E = toGroup(vip.GetStringMapString("groups.e2e"))

	params.NodeAddresses = vip.GetStringSlice("nodeAddresses")
	params.NodeIDs = vip.GetStringSlice("nodeIDs")
	params.Gateways = vip.GetStringSlice("gateways")
	params.RegServerPK = vip.GetString("reg_server_pk")

	return &params, nil
}
