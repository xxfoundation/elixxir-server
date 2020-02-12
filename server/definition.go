package server

import (
	"gitlab.com/elixxir/comms/connect"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/crypto/signature/rsa"
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
	PublicKey  *rsa.PublicKey
	PrivateKey *rsa.PrivateKey

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
	// Function to handle the wrapping-up of metrics for the first node
	MetricsHandler MetricsHandler

	//Size of the batch for the network
	BatchSize uint32
	//Network CMIX group
	CmixGroup *cyclic.Group
	//Network's End to End encryption group
	E2EGroup *cyclic.Group

	//Topology of the network as a whole
	Topology *connect.Circuit
	//Holds information about all other nodes in the network
	Nodes []Node
	//Holds information about the permissioning server
	Permissioning Perm

	// Generates random numbers
	RngStreamGen *fastRNG.StreamGenerator

	// timeout for round creation
	RoundCreationTimeout int
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
	PublicKey *rsa.PublicKey
	// IP address of the permissioning server
	Address string
	// Node Registration Code
	RegistrationCode string
}

type GW struct {
	// ID of the gateway
	ID *id.Gateway
	// PEM file containing the TLS cert
	TlsCert []byte
	// IP address of the gateway
	Address string
}

type MetricsHandler func(i *Instance, roundID id.Round) error
