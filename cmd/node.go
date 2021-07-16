///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

// Package node contains the initialization and main loop of a cMix server.
package cmd

import (
	"fmt"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/server/cmd/conf"
	"gitlab.com/elixxir/server/graphs"
	"gitlab.com/elixxir/server/internal"
	"gitlab.com/elixxir/server/internal/phase"
	"gitlab.com/elixxir/server/internal/state"
	"gitlab.com/elixxir/server/io"
	"gitlab.com/elixxir/server/node"
	"gitlab.com/elixxir/server/services"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/primitives/id"
	"os"
	"runtime"
	"strings"
	"time"
)

// StartServer reads configuration options and starts the cMix server
func StartServer(vip *viper.Viper) (*internal.Instance, error) {
	vip.Debug()

	jww.INFO.Printf("Log Filename: %v\n", vip.GetString("cmix.paths.log"))
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

	ps := fmt.Sprintf("Loaded params: %+v", params)
	ps = strings.ReplaceAll(ps,
		"Password:"+params.Database.Password,
		"Password:[dbpass]")
	ps = strings.ReplaceAll(ps,
		"RegistrationCode:"+params.RegistrationCode,
		"RegistrationCode:[regcode]")
	jww.INFO.Printf(ps)

	RecordPrivateKeyAndCertPaths(params.Node.Paths.Key,
		params.Node.Paths.Cert)

	jww.INFO.Printf("Converting params to server definition...")
	def, err := params.ConvertToDefinition()
	if err != nil {
		return nil, errors.Errorf("Failed to convert params to definition: %+v", err)
	}
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

	jww.INFO.Printf("Creating server instance...")

	ourChangeList := node.NewStateChanges()

	// Update the changelist to contain state functions
	ourChangeList[current.NOT_STARTED] = func(from current.Activity) error {
		return node.NotStarted(instance)
	}

	ourChangeList[current.WAITING] = func(from current.Activity) error {
		return node.Waiting(from)
	}

	ourChangeList[current.PRECOMPUTING] = func(from current.Activity) error {
		return node.Precomputing(instance)
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
		instance, err = internal.CreateServerInstance(def, io.NewImplementation, ourMachine, currentVersion)
		if err != nil {
			return instance, errors.Errorf("Could not create server instance: %v", err)
		}
	} else {
		// Otherwise, start in recovery mode
		jww.INFO.Println("Server has recovered from an error")
		instance, err = internal.RecoverInstance(def, io.NewImplementation, ourMachine, currentVersion)
		if err != nil {
			return instance, errors.WithMessage(err, "Could not recover server instance")
		}
	}

	if params.PhaseOverrides != nil {
		overrides := map[int]phase.Phase{}
		gc := services.NewGraphGenerator(4,
			uint8(runtime.NumCPU()), 1, 0)
		g := graphs.InitErrorGraph(gc)
		th := func(roundID id.Round, instance phase.GenericInstance,
			getChunk phase.GetChunk, getMessage phase.GetMessage) error {
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

	jww.INFO.Printf("~~~~~~~~~~~~~~~~~~~~~~~~\nServer Definition:\n%#v\n~~~~~~~~~~~~~~~~~~~~~~~~", def)

	// initialize the network
	instance.Online = true

	jww.INFO.Printf("Beginning resource queue...")
	//Begin the resource queue
	err = instance.Run()
	if err != nil {
		return instance, errors.Errorf("Unable to run instance: %+v", err)
	}

	return instance, nil
}
