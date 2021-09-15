///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package internal

import (
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/server/internal/measure"
	"gitlab.com/elixxir/server/services"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/ndf"
)

// Definition in cmd/node.go, it is filling this out
// polling is an ongoing process, and ..
// remove from this anything not about node
// move removed fields into comms network instance
// need to worry about nodes, gateways, perm
// nodes/gws have id's, add func in prim/ndf to get those
// integrate usage of network
// nodes/gws as id types
type Definition struct {
	// Holds input flags
	Flags

	//The ID of the node in the correct format
	ID *id.ID
	// The Salt used to generate the Node ID
	Salt []byte

	//DSA Keys defining the node's ownership
	PublicKey  *rsa.PublicKey
	PrivateKey *rsa.PrivateKey

	//PEM file containing the TLS cert
	TlsCert []byte
	//PEM file containing the TLS Key
	TlsKey []byte
	// The local address and port of this server
	ListeningAddress string
	// The public address and port of this server
	PublicAddress string
	// Interconnect port
	InterconnectPort int

	//Path the node will store its log at
	LogPath string
	//Path which system metrics are stored for first node
	MetricLogPath string

	// Path to save the list of node IP addresses
	IpListOutput string

	// Path where the Server and Gateway certificates will be stored
	ServerCertPath  string
	GatewayCertPath string
	//Information about the node's gateway
	Gateway GW

	// Information on permissioning server
	Network          Perm
	RegistrationCode string

	// Our NDFs for both backend servers and front-ends
	FullNDF    *ndf.NetworkDefinition
	PartialNDF *ndf.NetworkDefinition

	//Defines the properties of graphs in the node
	GraphGenerator services.GraphGenerator
	//Holds the ResourceMonitor object
	ResourceMonitor *measure.ResourceMonitor
	// Function to handle the wrapping-up of metrics for the first node
	MetricsHandler MetricsHandler

	// Generates random numbers
	RngStreamGen *fastRNG.StreamGenerator

	// timeout for round creation
	RoundCreationTimeout int

	// Toggles comm streaming
	DisableStreaming bool

	// Path for outputting errors to file for recovery
	RecoveredErrorPath string

	// Database parameters
	DbUsername string
	DbPassword string
	DbName     string
	DbAddress  string
	DbPort     string

	// Miscellaneous parameters
	DevMode       bool
	RawPermAddr   bool
	UsePrecanKeys bool
}

// Holds all input flags to the system.
type Flags struct {
	// Denotes if the server is to store all round keys indefinably
	KeepBuffers bool
	// If true, use GPU acceleration for precomputation
	UseGPU bool
	// If set, it should be used to overwrite the local address
	OverrideInternalIP string
}

//Holds information about another node in the network
type Node struct {
	// ID of the other node
	ID *id.ID
	// PEM file containing the TLS cert
	TlsCert []byte
	// IP of the []byte node
	Address string
}

// Perm Holds information about the permissioning server
type Perm struct {
	// PEM file containing the TLS cert
	TlsCert []byte
	// Public key used to sign valid client registrations
	PublicKey *rsa.PublicKey
	// IP address of the permissioning server
	Address string
}

type GW struct {
	// ID of the gateway
	ID *id.ID
	// PEM file containing the TLS cert
	TlsCert []byte
	// IP address of the gateway
	Address string
}

type MetricsHandler func(i *Instance, roundID id.Round) error
