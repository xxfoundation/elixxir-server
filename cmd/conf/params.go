///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package conf

import (
	gorsa "crypto/rsa"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"
	"gitlab.com/elixxir/comms/publicAddress"
	"gitlab.com/elixxir/crypto/cmix"
	"gitlab.com/elixxir/server/internal"
	"gitlab.com/elixxir/server/services"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/crypto/tls"
	"gitlab.com/xx_network/crypto/xx"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/idf"
	"gitlab.com/xx_network/primitives/ndf"
	"gitlab.com/xx_network/primitives/utils"
	"net"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// The default path to save the list of node IP addresses
const defaultIpListPath = "/opt/xxnetwork/node-logs/ipList.txt"

// This object is used by the server instance.
// It should be constructed using a viper object
type Params struct {
	KeepBuffers           bool
	UseGPU                bool
	OverrideInternalIP    string
	RngScalingFactor      uint `yaml:"rngScalingFactor"`
	SignedCertPath        string
	SignedGatewayCertPath string
	RegistrationCode      string

	Node          Node
	Database      Database
	Gateway       Gateway
	Permissioning Permissioning
	Metrics       Metrics
	GraphGen      GraphGen

	PhaseOverrides   []int
	OverrideRound    int
	RecoveredErrPath string

	DevMode bool
}

// NewParams gets elements of the viper object
// and updates the params object. It returns params
// unless it fails to parse in which it case returns error
func NewParams(vip *viper.Viper) (*Params, error) {

	var err error

	var require = func(s string, key string) {
		if s == "" {
			jww.FATAL.Panicf("%s must be set in params", key)
		}
	}

	params := Params{}

	params.RegistrationCode = vip.GetString("registrationCode")
	require(params.RegistrationCode, "registrationCode")

	params.Node.Port = vip.GetInt("node.Port")
	if params.Node.Port == 0 {
		jww.FATAL.Panic("Must specify a port to run on")
	}

	// Get server's public IP address or use the override IP, if set
	overridePublicIP := vip.GetString("node.overridePublicIP")
	params.Node.PublicAddress, err = publicAddress.GetIpOverride(overridePublicIP, params.Node.Port)
	if err != nil {
		jww.FATAL.Panicf("Failed to get public override IP \"%s\": %+v",
			overridePublicIP, err)
	}

	// Construct listening address; defaults to 0.0.0.0 if not set
	listeningIP := vip.GetString("node.listeningAddress")
	if listeningIP == "" {
		listeningIP = "0.0.0.0"
	}
	params.Node.ListeningAddress = net.JoinHostPort(listeningIP, strconv.Itoa(params.Node.Port))

	// Construct server's override internal IP address, if set
	overrideInternalIP := vip.GetString("node.overrideInternalIP")
	params.OverrideInternalIP, err = publicAddress.JoinIpPort(overrideInternalIP, params.Node.Port)
	if err != nil {
		jww.FATAL.Panicf("Failed to get public override IP \"%s\": %+v",
			overrideInternalIP, err)
	}

	params.Node.InterconnectPort = vip.GetInt("node.interconnectPort")

	params.Node.Paths.Idf = vip.GetString("node.paths.idf")
	require(params.Node.Paths.Idf, "node.paths.idf")

	params.Node.Paths.Cert = vip.GetString("node.paths.cert")
	require(params.Node.Paths.Cert, "node.paths.cert")

	params.Node.Paths.Key = vip.GetString("node.paths.key")
	require(params.Node.Paths.Key, "node.paths.key")

	params.Node.Paths.Log = vip.GetString("node.paths.log")
	if params.Node.Paths.Log == "" {
		params.Node.Paths.Log = "./node.log"
	}
	params.RecoveredErrPath = vip.GetString("node.paths.errOutput")
	require(params.RecoveredErrPath, "node.paths.errOutput")

	// If no path was supplied, then use the default
	params.Node.Paths.ipListOutput = vip.GetString("node.paths.ipListOutput")
	if params.Node.Paths.ipListOutput == "" {
		params.Node.Paths.ipListOutput = defaultIpListPath
	}

	// Obtain database connection info
	rawAddr := viper.GetString("database.address")
	var addr, port string
	if rawAddr != "" {
		addr, port, err = net.SplitHostPort(rawAddr)
		if err != nil {
			jww.FATAL.Panicf("Unable to get database port from %s: %+v", rawAddr, err)
		}
	}
	params.Database.Name = vip.GetString("database.name")
	params.Database.Username = vip.GetString("database.username")
	params.Database.Password = vip.GetString("database.password")
	params.Database.Address = addr
	params.Database.Port = port

	params.Gateway.Paths.Cert = vip.GetString("gateway.paths.cert")
	require(params.Gateway.Paths.Cert, "gateway.paths.cert")

	params.Permissioning.Paths.Cert = vip.GetString("permissioning.paths.cert")
	require(params.Permissioning.Paths.Cert, "permissioning.paths.cert")

	params.Permissioning.Address = vip.GetString("permissioning.address")
	require(params.Permissioning.Address, "permissioning.address")

	params.GraphGen.defaultNumTh = uint8(vip.GetUint("graphgen.defaultNumTh"))
	if params.GraphGen.defaultNumTh == 0 {
		params.GraphGen.defaultNumTh = uint8(runtime.NumCPU())
	}
	params.GraphGen.minInputSize = vip.GetUint32("graphgen.mininputsize")
	if params.GraphGen.minInputSize == 0 {
		params.GraphGen.minInputSize = 4
	}
	params.GraphGen.outputSize = vip.GetUint32("graphgen.outputsize")
	if params.GraphGen.outputSize == 0 {
		params.GraphGen.outputSize = 4
	}
	// This (outputThreshold) already defaulted to 0.0
	params.GraphGen.outputThreshold = float32(vip.GetFloat64("graphgen.outputthreshold"))

	params.KeepBuffers = vip.GetBool("keepBuffers")
	params.UseGPU = vip.GetBool("useGPU")
	params.RngScalingFactor = vip.GetUint("rngScalingFactor")
	// If RngScalingFactor is not set, then set default value
	if params.RngScalingFactor == 0 {
		params.RngScalingFactor = 10000
	}

	params.PhaseOverrides = vip.GetIntSlice("phaseOverrides")
	overrideRoundKey := "overrideRound"
	vip.SetDefault(overrideRoundKey, -1)
	params.OverrideRound = vip.GetInt(overrideRoundKey)

	params.Metrics.Log = vip.GetString("metrics.log")

	params.DevMode = viper.GetBool("devMode")

	return &params, nil
}

// Create a new Definition object from the Params object
func (p *Params) ConvertToDefinition() (*internal.Definition, error) {

	def := &internal.Definition{}

	def.Flags.KeepBuffers = p.KeepBuffers
	def.Flags.UseGPU = p.UseGPU
	def.RegistrationCode = p.RegistrationCode

	var tlsCert, tlsKey []byte
	var err error

	if p.Node.Paths.Cert != "" {
		tlsCert, err = utils.ReadFile(p.Node.Paths.Cert)

		if err != nil {
			jww.FATAL.Panicf("Could not load TLS Cert: %+v", err)
		}
	}

	if p.Node.Paths.Key != "" {
		tlsKey, err = utils.ReadFile(p.Node.Paths.Key)

		if err != nil {
			jww.FATAL.Panicf("Could not load TLS Key: %+v", err)
		}
	}

	def.ListeningAddress = p.Node.ListeningAddress
	def.PublicAddress = p.Node.PublicAddress
	def.InterconnectPort = p.Node.InterconnectPort
	def.TlsCert = tlsCert
	def.TlsKey = tlsKey
	def.LogPath = p.Node.Paths.Log
	def.MetricLogPath = p.Metrics.Log
	def.RecoveredErrorPath = p.RecoveredErrPath
	def.IpListOutput = p.Node.Paths.ipListOutput
	def.Flags.OverrideInternalIP = p.OverrideInternalIP
	def.DbUsername = p.Database.Username
	def.DbPassword = p.Database.Password
	def.DbName = p.Database.Name
	def.DbAddress = p.Database.Address
	def.DbPort = p.Database.Password

	if def.Flags.OverrideInternalIP != "" && !strings.Contains(def.Flags.OverrideInternalIP, ":") {
		def.Flags.OverrideInternalIP = net.JoinHostPort(def.Flags.OverrideInternalIP, strconv.Itoa(p.Node.Port))
	}

	var GwTlsCerts []byte

	if p.Gateway.Paths.Cert != "" {
		GwTlsCerts, err = utils.ReadFile(p.Gateway.Paths.Cert)
		if err != nil {
			jww.FATAL.Panicf("Could not load gateway TLS Cert: %+v", err)
		}
	}

	def.Gateway.TlsCert = GwTlsCerts

	var PermTlsCert []byte

	if p.Permissioning.Paths.Cert != "" {
		PermTlsCert, err = utils.ReadFile(p.Permissioning.Paths.Cert)

		if err != nil {
			jww.FATAL.Panicf("Could not load permissioning TLS Cert: %+v", err)
		}
	}

	//Set the node's private/public key
	var privateKey *rsa.PrivateKey
	var publicKey *rsa.PublicKey

	if p.Node.Paths.Cert != "" || p.Node.Paths.Key != "" {
		// Get the node's TLS cert
		tlsCertPEM, err := utils.ReadFile(p.Node.Paths.Cert)
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
		tlsKeyPEM, err := utils.ReadFile(p.Node.Paths.Key)
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

	// Check if the IDF exists
	if p.Node.Paths.Idf != "" && utils.Exists(p.Node.Paths.Idf) {
		// If the IDF exists, then get the ID and save it
		def.Salt, def.ID, err = idf.UnloadIDF(p.Node.Paths.Idf)
		if err != nil {
			return nil, errors.Errorf("Could not unload IDF: %+v", err)
		}
	} else {
		// If the IDF does not exist, then generate a new ID, save it to an IDF,
		// and save the ID to the definition

		// Generate a random 256-bit number for the salt
		def.Salt = cmix.NewSalt(csprng.NewSystemRNG(), 32)

		// Generate new ID
		newID, err2 := xx.NewID(def.PublicKey, def.Salt[:32], id.Node)
		if err2 != nil {
			return nil, errors.Errorf("Failed to create new ID: %+v", err2)
		}

		// Save new ID to file
		err2 = idf.LoadIDF(p.Node.Paths.Idf, def.Salt, newID)
		if err2 != nil {
			return nil, errors.Errorf("Failed to save new ID to file: %+v",
				err2)
		}

		def.ID = newID
	}

	def.Gateway.ID = def.ID.DeepCopy()
	def.Gateway.ID.SetType(id.Gateway)

	def.Permissioning.TlsCert = PermTlsCert
	def.Permissioning.Address = p.Permissioning.Address
	if len(def.Permissioning.TlsCert) > 0 {
		permCert, err := tls.LoadCertificate(string(def.Permissioning.TlsCert))
		if err != nil {
			jww.FATAL.Panicf("Could not decode permissioning tls cert file "+
				"into a tls cert: %v", err)
		}

		def.Permissioning.PublicKey = &rsa.PublicKey{PublicKey: *permCert.PublicKey.(*gorsa.PublicKey)}
	}

	//
	ourNdf := createNdf(def, p)
	def.FullNDF = ourNdf
	def.PartialNDF = ourNdf

	def.GraphGenerator = services.NewGraphGenerator(p.GraphGen.minInputSize,
		p.GraphGen.defaultNumTh, p.GraphGen.outputSize, p.GraphGen.outputThreshold)

	def.DevMode = p.DevMode
	return def, nil
}

// createNdf is a helper function which builds a network ndf based off of the
//  server.Definition
func createNdf(def *internal.Definition, params *Params) *ndf.NetworkDefinition {
	// Build our node
	ourNode := ndf.Node{
		ID:             def.ID.Marshal(),
		Address:        def.PublicAddress,
		TlsCertificate: string(def.TlsCert),
	}

	// Build our gateway
	ourGateway := ndf.Gateway{
		ID:             def.Gateway.ID.Marshal(),
		Address:        "0.0.0.0",
		TlsCertificate: string(def.Gateway.TlsCert),
	}

	// Build the perm server
	ourPerm := ndf.Registration{
		Address:        def.Permissioning.Address,
		TlsCertificate: string(def.Permissioning.TlsCert),
	}

	networkDef := &ndf.NetworkDefinition{
		Timestamp:    time.Time{},
		Gateways:     []ndf.Gateway{ourGateway},
		Nodes:        []ndf.Node{ourNode},
		Registration: ourPerm,
		Notification: ndf.Notification{},
		UDB:          ndf.UDB{ID: id.UDB.Marshal()},
	}

	return networkDef

}

// todo: docstring
func toNdfGroup(grp map[string]string) ndf.Group {
	pStr, pOk := grp["prime"]
	gStr, gOk := grp["generator"]

	if !gOk || !pOk {
		jww.FATAL.Panicf("Invalid Group Config "+
			"(prime: %v, generator: %v",
			pOk, gOk)
	}

	return ndf.Group{
		Prime:     pStr,
		Generator: gStr,
	}
}
