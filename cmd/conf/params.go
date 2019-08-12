////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package conf

import (
	gorsa "crypto/rsa"
	"encoding/base64"
	"fmt"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"
	"gitlab.com/elixxir/comms/utils"
	"gitlab.com/elixxir/crypto/signature/rsa"
	"gitlab.com/elixxir/crypto/tls"
	"gitlab.com/elixxir/primitives/circuit"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/server"
	"golang.org/x/crypto/blake2b"
	"io/ioutil"
	"net"
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
	Metrics       Metrics
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

	params.Metrics.Log = vip.GetString("metrics.log")

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
			if params.Index == i && len(params.Node.Id) == 0 {
				params.Node.Id = fakeId
			}
		}
	}

	return &params, nil
}

// Create a new Definition object from the Params object
func (p *Params) ConvertToDefinition() *server.Definition {

	def := &server.Definition{}

	def.Flags.KeepBuffers = p.KeepBuffers
	def.Flags.SkipReg = p.SkipReg
	def.Flags.Verbose = p.Verbose

	var tlsCert, tlsKey []byte
	var err error

	if p.Node.Paths.Cert != "" {
		tlsCert, err = ioutil.ReadFile(utils.GetFullPath(p.Node.Paths.Cert))

		if err != nil {
			jww.FATAL.Panicf("Could not load TLS Cert: %+v", err)
		}
	}

	if p.Node.Paths.Key != "" {
		tlsKey, err = ioutil.ReadFile(utils.GetFullPath(p.Node.Paths.Key))

		if err != nil {
			jww.FATAL.Panicf("Could not load TLS Key: %+v", err)
		}
	}

	ids := p.Node.Ids
	var nodes []server.Node
	var nodeIDs []*id.Node

	var nodeIDDecodeErrorHappened bool
	for i, currId := range ids {
		nodeID, err := base64.StdEncoding.DecodeString(currId)
		jww.INFO.Printf("Creating Def for Node ID: %s", nodeID)
		if err != nil {
			// This indicates a server misconfiguration which needs fixing for
			// the server to function properly
			err = errors.Wrapf(err, "Node ID at index %v failed to decode", i)
			jww.ERROR.Print(err)
			nodeIDDecodeErrorHappened = true
		}
		n := server.Node{
			ID:      id.NewNodeFromBytes(nodeID),
			TlsCert: tlsCert,
			Address: p.Node.Addresses[i],
		}
		nodes = append(nodes, n)
		nodeIDs = append(nodeIDs, id.NewNodeFromBytes(nodeID))
	}

	if nodeIDDecodeErrorHappened {
		jww.FATAL.Panic("One or more node IDs didn't base64 decode correctly")
	}

	nodeID, err := base64.StdEncoding.DecodeString(p.Node.Id)
	if err != nil {
		// This indicates a server misconfiguration which needs fixing for
		// the server to function properly
		err = errors.Wrapf(err, "Node ID failed to decode")
		jww.ERROR.Print(err)
		nodeIDDecodeErrorHappened = true
	}
	def.ID = id.NewNodeFromBytes(nodeID)

	_, port, err := net.SplitHostPort(p.Node.Addresses[p.Index])
	if err != nil {
		jww.FATAL.Panicf("Unable to obtain port from address: %+v",
			errors.New(err.Error()))
	}
	def.Address = fmt.Sprintf("0.0.0.0:%s", port)
	def.TlsCert = tlsCert
	def.TlsKey = tlsKey
	def.LogPath = p.Node.Paths.Log
	def.MetricLogPath = p.Metrics.Log
	def.Gateway.Address = p.Gateways.Addresses[p.Index]
	var GwTlsCerts []byte

	if p.Gateways.Paths.Cert != "" {
		GwTlsCerts, err = ioutil.ReadFile(utils.GetFullPath(p.Gateways.Paths.Cert))
		if err != nil {
			jww.FATAL.Panicf("Could not load gateway TLS Cert: %+v", err)
		}
	}

	def.Gateway.TlsCert = GwTlsCerts
	def.Gateway.ID = def.ID.NewGateway()
	def.BatchSize = p.Batch
	def.CmixGroup = p.Groups.GetCMix()
	def.E2EGroup = p.Groups.GetE2E()

	def.Topology = circuit.New(nodeIDs)
	def.Nodes = nodes

	var PermTlsCert []byte

	if p.Permissioning.Paths.Cert != "" {
		PermTlsCert, err = ioutil.ReadFile(utils.GetFullPath(p.Permissioning.Paths.Cert))

		if err != nil {
			jww.FATAL.Panicf("Could not load permissioning TLS Cert: %+v", err)
		}
	}

	//Set the node's private/public key
	var privateKey *rsa.PrivateKey
	var publicKey *rsa.PublicKey

	if p.Node.Paths.Cert == "" || p.Node.Paths.Key == "" {
		jww.FATAL.Panicf("Could not generate RSA key: %+v", err)
	} else {
		// Get the node's TLS cert
		tlsCertPEM, err := ioutil.ReadFile(p.Node.Paths.Cert)
		if err != nil {
			jww.FATAL.Panicf("Could not read tls cert file: %v", err)
		}

		//Get the RSA key out of the TLS cert
		tlsCert, err := tls.LoadCertificate(string(tlsCertPEM))
		if err != nil {
			jww.FATAL.Panicf("Could not decode tls cert file into a"+
				" tls cert: %v", err)
		}

		publicKey = &rsa.PublicKey{PublicKey: *tlsCert.PublicKey.(*gorsa.PublicKey)}

		// Get the node's TLS Key
		tlsKeyPEM, err := ioutil.ReadFile(p.Node.Paths.Key)
		if err != nil {
			jww.FATAL.Panicf("Could not read tls key file: %v", err)
		}

		privateKey, err = rsa.LoadPrivateKeyFromPem(tlsKeyPEM)
		if err != nil {
			jww.FATAL.Panicf("Could not decode tls key from file: %+v",
				err)
		}
	}

	def.PublicKey = publicKey
	def.PrivateKey = privateKey

	def.Permissioning.TlsCert = PermTlsCert
	def.Permissioning.Address = p.Permissioning.Address
	def.Permissioning.RegistrationCode = p.Permissioning.RegistrationCode

	permCert, err := tls.LoadCertificate(string(def.Permissioning.TlsCert))
	if err != nil {
		jww.FATAL.Panicf("Could not decode permissioning tls cert file "+
			"into a tls cert: %v", err)
	}

	publicKey = &rsa.PublicKey{PublicKey: *permCert.PublicKey.(*gorsa.PublicKey)}

	return def
}
