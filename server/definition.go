package server

import (
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/signature"
	"gitlab.com/elixxir/primitives/circuit"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/globals"
	"gitlab.com/elixxir/server/server/measure"
	"gitlab.com/elixxir/server/services"
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
	TLS_Cert []byte
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
