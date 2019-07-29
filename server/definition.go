package server

import (
	"encoding/base64"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/signature"
	"gitlab.com/elixxir/primitives/circuit"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/cmd/conf"
	"gitlab.com/elixxir/server/globals"
	"gitlab.com/elixxir/server/server/measure"
	"gitlab.com/elixxir/server/services"
	"io/ioutil"
)

type Definition struct {
	// Holds input flags
	Flags

	//The ID of the node in the correct format
	ID *id.Node

	//DSA Keys defining the node's ownership
	DsaPublicKey  *signature.DSAPublicKey
	DsaPrivateKey *signature.DSAPrivateKey

	//PEM file containing the TLS cert
	TlsCert []byte
	//PEM file containing the TLS Key
	TlsKey []byte
	//String containing the local address and port to connect to
	Address string

	//Path the node will store its log at
	LogPath string
	//Path which system metrics are stored for first node
	MetricLogPath string

	//Information about the node's gateway
	Gateway GW

	//Links to the database holding user keys
	UserRegistry globals.UserRegistry
	//Defines the properties of graphs in the node
	GraphGenerator services.GraphGenerator
	//Holds the ResourceMonitor object
	ResourceMonitor *measure.ResourceMonitor

	//Size of the batch for the network
	BatchSize uint32
	//Network CMIX group
	CmixGroup *cyclic.Group
	//Network's End to End encryption group
	E2EGroup *cyclic.Group

	//Topology of the network as a whole
	Topology *circuit.Circuit
	//Holds information about all other nodes in the network
	Nodes []Node
	//Holds information about the permissioning server
	Permissioning Perm
}

// Holds all input flags to the system.
type Flags struct {
	// Starts a server without client registration
	SkipReg bool
	// Prints all logs
	Verbose bool
	// Denotes if the server is to store all round keys indefinably
	KeepBuffers bool
}

//Holds information about another node in the network
type Node struct {
	// ID of the other node
	ID *id.Node
	// PEM file containing the TLS cert
	TlsCert []byte
	// IP of the []byte node
	Address string
}

// Holds information about the permissioning server
type Perm struct {
	// PEM file containing the TLS cert
	TlsCert []byte
	// Public key used to sign valid client registrations
	DsaPublicKey *signature.DSAPublicKey
	// IP address of the permissioning server
	Address string
}

type GW struct {
	// ID of the gateway
	ID *id.Gateway
	// PEM file containing the TLS cert
	TlsCert []byte
	// IP address of the gateway
	Address string
}

// Create a new Definition object from the given parameters
func NewDefinition(params *conf.Params, pub *signature.DSAPublicKey,
	priv *signature.DSAPrivateKey) *Definition {

	def := &Definition{}

	def.Flags.KeepBuffers = params.KeepBuffers
	def.Flags.SkipReg = params.SkipReg
	def.Flags.Verbose = params.Verbose

	var tlsCert, tlsKey []byte
	var err error

	if params.Node.Paths.Cert != "" {
		tlsCert, err = ioutil.ReadFile(params.Node.Paths.Cert)

		if err != nil {
			jww.FATAL.Panicf("Could not load TLS Cert: %+v", err)
		}
	}

	if params.Node.Paths.Key != "" {
		tlsKey, err = ioutil.ReadFile(params.Node.Paths.Key)

		if err != nil {
			jww.FATAL.Panicf("Could not load TLS Key: %+v", err)
		}
	}

	ids := params.Node.Ids
	nodes := make([]Node, len(ids))
	nodeIDs := make([]*id.Node, len(ids))

	var nodeIDDecodeErrorHappened bool
	for i := range ids {
		nodeID, err := base64.StdEncoding.DecodeString(ids[i])
		if err != nil {
			// This indicates a server misconfiguration which needs fixing for
			// the server to function properly
			err = errors.Wrapf(err, "Node ID at index %v failed to decode", i)
			jww.ERROR.Print(err)
			nodeIDDecodeErrorHappened = true
		}
		n := Node{
			ID:      id.NewNodeFromBytes(nodeID),
			TlsCert: tlsCert,
			Address: ids[i],
		}
		nodes = append(nodes, n)
		nodeIDs = append(nodeIDs, id.NewNodeFromBytes(nodeID))
	}
	if nodeIDDecodeErrorHappened {
		jww.FATAL.Panic("One or more node IDs didn't base64 decode correctly")
	}

	def.ID = nodes[params.Index].ID
	def.Address = nodes[params.Index].Address
	def.TlsCert = tlsCert
	def.TlsKey = tlsKey

	def.LogPath = params.Node.Paths.Log
	def.MetricLogPath = params.Metrics.Log

	def.Gateway.Address = params.Gateways.Addresses[params.Index]

	var GwTlsCerts []byte

	if params.Gateways.Paths.Cert != "" {
		GwTlsCerts, err = ioutil.ReadFile(params.Gateways.Paths.Cert)

		if err != nil {
			jww.FATAL.Panicf("Could not load gateway TLS Cert: %+v", err)
		}
	}

	def.Gateway.TlsCert = GwTlsCerts
	def.Gateway.ID = def.ID.NewGateway()

	def.BatchSize = params.Batch
	def.CmixGroup = params.Groups.GetCMix()
	def.E2EGroup = params.Groups.GetE2E()

	def.Topology = circuit.New(nodeIDs)
	def.Nodes = nodes

	def.DsaPrivateKey = priv
	def.DsaPublicKey = pub

	var PermTlsCert []byte

	if params.Permissioning.Paths.Cert != "" {
		tlsCert, err = ioutil.ReadFile(params.Permissioning.Paths.Cert)

		if err != nil {
			jww.FATAL.Panicf("Could not load permissioning TLS Cert: %+v", err)
		}
	}

	def.Permissioning.TlsCert = PermTlsCert
	def.Permissioning.Address = params.Permissioning.Address

	def.Permissioning.DsaPublicKey = &signature.DSAPublicKey{}
	def.Permissioning.DsaPublicKey, err = def.Permissioning.DsaPublicKey.
		PemDecode([]byte(params.Permissioning.PublicKey))
	if err != nil {
		jww.FATAL.Panicf("Unable to decode permissioning public key: %+v", err)
	}

	return def
}
