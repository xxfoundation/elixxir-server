////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package conf

import (
	"encoding/base64"
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"
	"golang.org/x/crypto/blake2b"
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
	params.Permissioning.PublicKey = vip.GetString("permissioning.publicKey")

	params.Batch = vip.GetUint32("batch")
	params.SkipReg = vip.GetBool("skipReg")
	params.Verbose = vip.GetBool("verbose")
	params.KeepBuffers = vip.GetBool("keepBuffers")

	params.Groups.CMix = vip.GetStringMapString("groups.cmix")
	params.Groups.E2E = vip.GetStringMapString("groups.e2e")

	// In the event IDs are not able to be provided,
	// we can hash the node addresses as a workaround
	if len(params.Node.Ids) == 0 {
		hash, err := blake2b.New256(nil)
		if err != nil {
			jww.FATAL.Panicf("Unable to create ID hash %v", err)
		}

		jww.WARN.Printf("No Node IDs found, " +
			"generating from hash of Node address...")

		for i, addr := range params.Node.Addresses {
			hash.Reset()
			hash.Write([]byte(addr))
			fakeId := base64.StdEncoding.EncodeToString(hash.Sum(nil))
			params.Node.Ids = append(params.Node.Ids, fakeId)
			if params.Index == i {
				params.Node.Id = fakeId
			}
		}
	}

	return &params, nil
}
