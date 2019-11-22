////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Package node contains the initialization and main loop of a cMix server.
package cmd

import (
	"fmt"
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"
	"gitlab.com/elixxir/comms/connect"
	"gitlab.com/elixxir/crypto/csprng"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/cmd/conf"
	"gitlab.com/elixxir/server/globals"
	"gitlab.com/elixxir/server/io"
	"gitlab.com/elixxir/server/node"
	"gitlab.com/elixxir/server/permissioning"
	"gitlab.com/elixxir/server/server"
	"runtime"
	"time"
)

// Number of hard-coded users to create
var numDemoUsers = int(256)

// StartServer reads configuration options and starts the cMix server
func StartServer(vip *viper.Viper) {
	vip.Debug()

	jww.INFO.Printf("Log Filename: %v\n", vip.GetString("node.paths.log"))
	jww.INFO.Printf("Config Filename: %v\n", vip.ConfigFileUsed())

	//Set the max number of processes
	runtime.GOMAXPROCS(runtime.NumCPU() * 2)

	//Start the performance monitor
	resourceMonitor := monitorMemoryUsage(performanceCheckPeriod,
		deltaMemoryThreshold, minMemoryTrigger)

	// Load params object from viper conf
	params, err := conf.NewParams(vip)
	if err != nil {
		jww.FATAL.Println("Unable to load params from viper")
	}

	jww.INFO.Printf("Loaded params: %+v", params)

	//Check that there is a gateway
	if len(params.Gateways.Addresses) < 1 {
		// No gateways in config file or passed via command line
		jww.FATAL.Panicf("Error: No gateways specified! Add to" +
			" configuration file!")
		return
	}

	// Initialize the backend
	jww.INFO.Printf("Initalizing the backend")
	dbAddress := params.Database.Addresses[params.Index]
	cmixGrp := params.Groups.GetCMix()

	// Initialize the global group
	globals.SetGroup(cmixGrp)

	//Initialize the user database
	userDatabase := globals.NewUserRegistry(
		params.Database.Username,
		params.Database.Password,
		params.Database.Name,
		dbAddress,
	)

	//populate the dummy precanned users
	jww.INFO.Printf("Adding dummy users to registry")
	PopulateDummyUsers(userDatabase, cmixGrp)

	//Add a dummy user for gateway
	dummy := userDatabase.NewUser(cmixGrp)
	dummy.ID = id.MakeDummyUserID()
	dummy.BaseKey = cmixGrp.NewIntFromBytes((*dummy.ID)[:])
	dummy.IsRegistered = true
	userDatabase.UpsertUser(dummy)

	jww.INFO.Printf("Converting params to server definition")
	def := params.ConvertToDefinition()
	def.UserRegistry = userDatabase
	def.ResourceMonitor = resourceMonitor

	err = node.ClearMetricsLogs(def.MetricLogPath)
	if err != nil {
		jww.ERROR.Printf("Error deleting old metric log files: %v", err)
	}

	def.MetricsHandler = func(instance *server.Instance, roundID id.Round) error {
		return node.GatherMetrics(instance, roundID, metricsWhitespace)
	}

	PanicHandler := func(g, m string, err error) {
		jww.FATAL.Panicf(fmt.Sprintf("Error in module %s of graph %s: %+v", g,
			m, err))
	}
	def.GraphGenerator.SetErrorHandler(PanicHandler)

	def.RngStreamGen = fastRNG.NewStreamGenerator(params.RngScalingFactor,
		uint(runtime.NumCPU()), csprng.NewSystemRNG)

	if !disablePermissioning {
		// Blocking call: Begin Node registration
		nodes, nodeIds, serverCert, gwCert := permissioning.RegisterNode(def)
		def.Nodes = nodes
		def.TlsCert = []byte(serverCert)
		def.Gateway.TlsCert = []byte(gwCert)
		def.Topology = connect.NewCircuit(nodeIds)
	}

	jww.INFO.Printf("Creating server instance")
	// Create instance
	if noTLS {
		jww.INFO.Println("Blanking TLS certs for non use")
		def.TlsKey = nil
		def.TlsCert = nil
		def.Gateway.TlsCert = nil
		for i := 0; i < def.Topology.Len(); i++ {
			def.Nodes[i].TlsCert = nil
		}
	}
	fmt.Println("~~~~~~~~~~~~~~~~~~~~~~~~")
	fmt.Printf("Server Definition: \n%#v", def)
	fmt.Println("~~~~~~~~~~~~~~~~~~~~~~~~")
	instance, err := server.CreateServerInstance(def, node.NewImplementation)
	if err != nil {
		jww.FATAL.Printf("Could not create server instance: %v", err)
	}

	if instance.IsFirstNode() {
		jww.INFO.Printf("Initilizing as first node")
		instance.InitFirstNode()
	}
	if instance.IsLastNode() {
		jww.INFO.Printf("Initilizing as last node")
		instance.InitLastNode()
	}

	jww.INFO.Printf("Connecting to network")

	// if permissioning check that the certs are valid
	if !disablePermissioning {
		err = instance.VerifyTopology()
		if err != nil {
			jww.FATAL.Panicf("Could not verify all nodes were signed by the"+
				" permissioning server: %+v", err)
		}
	}

	// initialize the network
	instance.Online = true

	jww.INFO.Printf("Checking all servers are online")
	io.VerifyServersOnline(instance.GetNetwork(), instance.GetTopology())

	jww.INFO.Printf("Begining resource queue")
	//Begin the resource queue
	instance.Run()

	//Start runners for first node
	if instance.IsFirstNode() {
		jww.INFO.Printf("Starting first node network manager")
		instance.RunFirstNode(instance, roundBufferTimeout*time.Second,
			io.TransmitCreateNewRound, node.MakeStarter(params.Batch))
	}

}

// Create dummy users to be manually inserted into the database
func PopulateDummyUsers(ur globals.UserRegistry, grp *cyclic.Group) {
	// Deterministically create named users for demo
	for i := 1; i < numDemoUsers; i++ {
		u := ur.NewUser(grp)
		u.IsRegistered = true
		ur.UpsertUser(u)
	}
}
