////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Package node contains the initialization and main loop of a cMix server.
package cmd

import (
	"fmt"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"
	"gitlab.com/elixxir/crypto/csprng"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/cmd/conf"
	"gitlab.com/elixxir/server/globals"
	"gitlab.com/elixxir/server/graphs"
	"gitlab.com/elixxir/server/internal"
	"gitlab.com/elixxir/server/internal/phase"
	"gitlab.com/elixxir/server/internal/state"
	"gitlab.com/elixxir/server/io"
	"gitlab.com/elixxir/server/node"
	"gitlab.com/elixxir/server/services"
	"os"
	"runtime"
	"time"
)

// Number of hard-coded users to create
var numDemoUsers = int(256)

// StartServer reads configuration options and starts the cMix server
func StartServer(vip *viper.Viper) error {
	vip.Debug()

	jww.INFO.Printf("Log Filename: %v\n", vip.GetString("node.paths.log"))
	jww.INFO.Printf("Config Filename: %v\n", vip.ConfigFileUsed())

	//Set the max number of processes
	runtime.GOMAXPROCS(maxProcsOverride)

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
	if len(params.Gateway.Address) < 1 {
		// No gateways in config file or passed via command line
		return errors.New("Error: No gateway specified! Add to" +
			" configuration file!")
	}

	// Initialize the backend
	jww.INFO.Printf("Initalizing the backend")
	dbAddress := params.Database.Address
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
	dummy.ID = &id.DummyUser
	dummy.BaseKey = cmixGrp.NewIntFromBytes((*dummy.ID)[:])
	dummy.IsRegistered = true
	userDatabase.UpsertUser(dummy)

	jww.INFO.Printf("Converting params to server definition")
	def, err := params.ConvertToDefinition()
	if err != nil {
		return errors.Errorf("Failed to convert params to definition: %+v", err)
	}
	def.UserRegistry = userDatabase
	def.ResourceMonitor = resourceMonitor

	err = node.ClearMetricsLogs(def.MetricLogPath)
	if err != nil {
		jww.ERROR.Printf("Error deleting old metric log files: %v", err)
	}

	def.MetricsHandler = func(instance *internal.Instance, roundID id.Round) error {
		return node.GatherMetrics(instance, roundID, metricsWhitespace)
	}

	var instance *internal.Instance

	PanicHandler := node.GetDefaultPanicHanlder(instance)

	def.GraphGenerator.SetErrorHandler(PanicHandler)

	def.RngStreamGen = fastRNG.NewStreamGenerator(params.RngScalingFactor,
		uint(runtime.NumCPU()), csprng.NewSystemRNG)

	jww.INFO.Printf("Creating server instance")

	ourChangeList := node.NewStateChanges()

	// Update the changelist to contain state functions
	ourChangeList[current.NOT_STARTED] = func(from current.Activity) error {
		return node.NotStarted(instance, noTLS)
	}

	ourChangeList[current.WAITING] = func(from current.Activity) error {
		return node.Waiting(from)
	}

	ourChangeList[current.PRECOMPUTING] = func(from current.Activity) error {
		// todo: ask reviewer about this magic number
		return node.Precomputing(instance, 5*time.Second)
	}

	ourChangeList[current.STANDBY] = func(from current.Activity) error {
		return node.Standby(from)
	}

	ourChangeList[current.REALTIME] = func(from current.Activity) error {
		return node.Realtime(instance)
	}

	ourChangeList[current.COMPLETED] = func(from current.Activity) error {
		return node.Completed(from)
	}

	ourChangeList[current.ERROR] = func(from current.Activity) error {
		return node.Error(instance)
	}

	ourChangeList[current.CRASH] = func(from current.Activity) error {
		return node.Crash(from)
	}

	// Create the machine with these state functions
	ourMachine := state.NewMachine(ourChangeList)

	// Create instance
	recoveredErrorFile, err := os.Open(params.RecoveredErrFile)
	if err != nil {
		if os.IsNotExist(err) {
			instance, err = internal.CreateServerInstance(def, io.NewImplementation, ourMachine, noTLS, currentVersion)
			if err != nil {
				return errors.Errorf("Could not create server instance: %v", err)
			}
		} else {
			return errors.WithMessage(err, "Failed to open file")
		}
	} else {
		jww.INFO.Println("Server has recovered from an error")
		instance, err = internal.RecoverInstance(def, io.NewImplementation, ourMachine, noTLS, currentVersion, recoveredErrorFile)
		if err != nil {
			return errors.WithMessage(err, "Could not recover server instance")
		}
	}

	if params.PhaseOverrides != nil {
		overrides := map[int]phase.Phase{}
		gc := services.NewGraphGenerator(4, node.GetDefaultPanicHanlder(instance),
			uint8(runtime.NumCPU()), 1, 0)
		g := graphs.InitErrorGraph(gc)
		th := func(roundID id.Round, instance phase.GenericInstance, getChunk phase.GetChunk, getMessage phase.GetMessage) error {
			return errors.New("Failed intentionally")
		}
		for _, i := range params.PhaseOverrides {
			jww.ERROR.Println(fmt.Sprintf("Overriding phase %d", i))
			p := phase.New(phase.Definition{
				Graph:               g,
				Type:                phase.Type(i),
				TransmissionHandler: th,
				Timeout:             500,
				DoVerification:      false,
			})
			overrides[i] = p
		}
		if params.OverrideRound != -1 {
			instance.OverridePhasesAtRound(overrides, params.OverrideRound)
		} else {
			instance.OverridePhases(overrides)
		}
	}

	jww.INFO.Printf("Instance created!")

	// Create instance
	if noTLS {
		jww.INFO.Println("Blanking TLS certs for non use")
		def.TlsKey = nil
		def.TlsCert = nil
		def.Gateway.TlsCert = nil
		//for i := 0; i < def.Topology.Len(); i++ {
		//	def.Nodes[i].TlsCert = nil
		//}
	}
	fmt.Println("~~~~~~~~~~~~~~~~~~~~~~~~")
	fmt.Printf("Server Definition: \n%#v", def)
	fmt.Println("~~~~~~~~~~~~~~~~~~~~~~~~")
	def.RoundCreationTimeout = newRoundTimeout

	jww.INFO.Printf("Connecting to network")

	// initialize the network
	instance.Online = true

	jww.INFO.Printf("Begining resource queue")
	//Begin the resource queue
	err = instance.Run()
	if err != nil {
		return errors.Errorf("Unable to run instance: %+v", err)
	}

	jww.INFO.Printf("Checking all servers are online")

	return nil
}

// Create dummy users to be manually inserted into the database
func PopulateDummyUsers(ur globals.UserRegistry, grp *cyclic.Group) {
	// Deterministically create named users for demo
	for i := 1; i < numDemoUsers; i++ {
		u := ur.NewUser(grp)
		u.IsRegistered = true
		ur.UpsertUser(u)
	}
	return
}
