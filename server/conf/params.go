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

//node := Node{
//	Id: vip.GetString("node.id"),
//	Paths: Paths{
//		Cert: vip.GetString("node.paths.cert"),
//		Key:  vip.GetString("node.paths.key"),
//		Log:  vip.GetString("node.paths.log"),
//	},
//}
//
//database := Database{
//	Name:      vip.GetString("database.name"),
//	Username:  vip.GetString("database.username"),
//	Password:  vip.GetString("database.password"),
//	Addresses: vip.GetStringSlice("database.addresses"),
//}
//
//gateways := Gateways{
//	Paths: Paths{
//		Cert: vip.GetString("gateways.paths.cert"),
//	},
//	Addresses: vip.GetStringSlice("gateways.addresses"),
//}
//
//permissioning := Permissioning{
//	Paths: Paths{
//		Cert: vip.GetString("permissioning.paths.cert"),
//	},
//	Address: vip.GetString("permissioning.address"),
//	Regcode: vip.GetString("permissioning.registrationCode"),
//}
//
//global := Global{
//	Batch:   vip.GetUint32("global.batch"),
//	Skipreg: vip.GetBool("global.skipReg"),
//	Groups: Groups{
//		ExpectedGroup: toGroup(vip.GetStringMapString("permissioning.groups.cmix")),
//		E2E:  toGroup(vip.GetStringMapString("permissioning.groups.e2e")),
//	},
//}
//
//params := Params{
//	Node:          node,
//	Database:      database,
//	Gateways:      gateways,
//	Permissioning: permissioning,
//	Global:        global,
//}

//params.Node.Id = vip.GetString("node.id")
//params.Node.Paths =
//params.Node.Paths = vip.GetInt("index")

//params.Database.Name = vip.GetString("database.name")
//params.Database.Username = vip.GetString("database.username")
//params.Database.Password = vip.GetString("database.password")
//params.Database.Addresses = vip.GetStringSlice("database.addresses")
//
//params.Skipreg = vip.GetBool("skipReg")
//
//params.Path.Cert = vip.GetString("path.cert")
//params.Path.GatewayCert = vip.GetString("path.gateway_cert")
//params.Path.Key = vip.GetString("path.key")
//params.Path.Log = vip.GetString("path.log")
//
//params.Batch = vip.GetUint32("batch")
//params.Groups.ExpectedGroup = toGroup(vip.GetStringMapString("groups.cmix"))
//params.Groups.E2E = toGroup(vip.GetStringMapString("groups.e2e"))
//
//params.NodeAddresses = vip.GetStringSlice("nodeAddresses")
//params.NodeIDs = vip.GetStringSlice("nodeIDs")
//params.Gateways = vip.GetStringSlice("gateways")
//params.RegServerPK = vip.GetString("reg_server_pk")

//return &params, nil

//
//Skipreg  bool
//Path     Paths

////Network Identity Params
//Batch         uint32
//Groups        Groups
//RegServerPK   string
//NodeAddresses []string
//// these are base64 strings, so instance creation must base64 decode these
//// before using them as node IDs
//NodeIDs  []string
