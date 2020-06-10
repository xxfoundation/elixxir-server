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
		jww.FATAL.Panicf("Unable to load params from viper: %+v", err)
	}

	jww.INFO.Printf("Loaded params: %+v", params)

	// Initialize the backend
	jww.INFO.Printf("Initalizing the backend")
	dbAddress := params.Database.Address

	//Initialize the user database
	userDatabase := globals.NewUserRegistry(
		params.Database.Username,
		params.Database.Password,
		params.Database.Name,
		dbAddress,
	)

	jww.INFO.Printf("Converting params to server definition")
	def, err := params.ConvertToDefinition()
	if err != nil {
		return errors.Errorf("Failed to convert params to definition: %+v", err)
	}
	def.UserRegistry = userDatabase
	def.ResourceMonitor = resourceMonitor

	def.DisableStreaming = disableStreaming

	err = node.ClearMetricsLogs(def.MetricLogPath)
	if err != nil {
		jww.ERROR.Printf("Error deleting old metric log files: %v", err)
	}

	def.MetricsHandler = func(instance *internal.Instance, roundID id.Round) error {
		return node.GatherMetrics(instance, roundID)
	}

	var instance *internal.Instance

	def.RngStreamGen = fastRNG.NewStreamGenerator(params.RngScalingFactor,
		uint(runtime.NumCPU()), csprng.NewSystemRNG)

	jww.INFO.Printf("Creating server instance")

	ourChangeList := node.NewStateChanges()

	// Update the changelist to contain state functions
	ourChangeList[current.NOT_STARTED] = func(from current.Activity) error {
		return node.NotStarted(instance)
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

	// Check if the error recovery file exists
	if _, err := os.Stat(params.RecoveredErrPath); os.IsNotExist(err) {
		// If not, start normally
		instance, err = internal.CreateServerInstance(def,
			io.NewImplementation, ourMachine, useGPU, currentVersion)
		if err != nil {
			return errors.Errorf("Could not create server instance: %v", err)
		}
	} else {
		// Otherwise, start in recovery mode
		jww.INFO.Println("Server has recovered from an error")
		instance, err = internal.RecoverInstance(def, io.NewImplementation,
			ourMachine, useGPU, currentVersion)
		if err != nil {
			return errors.WithMessage(err, "Could not recover server instance")
		}
	}

	if params.PhaseOverrides != nil {
		overrides := map[int]phase.Phase{}
		gc := services.NewGraphGenerator(4,
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
				Timeout:             1 * time.Minute,
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

	fmt.Println("~~~~~~~~~~~~~~~~~~~~~~~~")
	fmt.Printf("Server Definition: \n%#v", def)
	fmt.Println("~~~~~~~~~~~~~~~~~~~~~~~~")

	jww.INFO.Printf("Connecting to network")

	// initialize the network
	instance.Online = true

	jww.INFO.Printf("Begining resource queue")
	//Begin the resource queue
	err = instance.Run()
	if err != nil {
		return errors.Errorf("Unable to run instance: %+v", err)
	}

	return nil
}
