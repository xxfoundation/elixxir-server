////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Package node contains the initialization and main loop of a cMix server.
package cmd

import (
	"fmt"
	//"encoding/binary"
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"
	"gitlab.com/elixxir/server/server/conf"

	//"gitlab.com/elixxir/comms/connect"
	//"gitlab.com/elixxir/comms/node"
	//"gitlab.com/elixxir/crypto/cyclic"
	//"gitlab.com/elixxir/primitives/id"
	//	"gitlab.com/elixxir/server/cryptops/realtime"
	//"gitlab.com/elixxir/server/globals"
	//"gitlab.com/elixxir/server/io"
	"runtime"
	"strings"
	//"sync/atomic"
	//"time"
)

// StartServer reads configuration options and starts the cMix server
func StartServer(vip *viper.Viper) {
	vip.Debug()

	jww.INFO.Printf("Log Filename: %v\n", vip.GetString("logPath"))
	jww.INFO.Printf("Config Filename: %v\n", vip.ConfigFileUsed())

	//Set the max number of processes
	runtime.GOMAXPROCS(runtime.NumCPU() * 2)

	//Start the performance monitor
	go MonitorMemoryUsage()

	// Load params object from viper conf
	params, err := conf.NewParams(vip)
	if err != nil {
		jww.FATAL.Println("Unable to load params from viper")
	}

	fmt.Printf("conf: %v\n", params.Index)

	//Check that there is a gateway
	if len(params.Gateways) < 1 {
		// No gateways in config file or passed via command line
		jww.FATAL.Panicf("Error: No gateways specified! Add to" +
			" configuration file!")
		return
	}

	//get the index of the server

	// Initialize the backend
	//dbAddress := params.Database.Addresses[serverIdx]

	/*users := globals.NewUserRegistry(
		params.Database.Username,
		params.Database.Password,
		params.Database.Name,
		dbAddress,
	)*/

	// Load group from viper
	// TODO: when you go back to hook up the new round/DSA stuff to main,
	// these should be assigned variables in there.
	jww.INFO.Printf("%v", viper.GetStringMapString(
		"cryptographicParameters.cMix"))
	grp := params.Groups.CMix
	e2eGrp := params.Groups.E2E
	// TODO: Add a Stringer interface to cyclic.Group
	jww.INFO.Printf("cMix Group: %d", grp.GetFingerprint())
	jww.INFO.Printf("E2E Group: %d", e2eGrp.GetFingerprint())

	jww.INFO.Print("Server list: " + strings.Join(params.NodeAddresses, ","))

	// Start mix servers on localServer
	localServer := params.NodeAddresses[serverIdx]
	jww.INFO.Printf("Starting server on %v\n", localServer)

	// ensure that the Node ID is populated
	//	viperNodeID := uint64(viper.GetInt("nodeid"))
	//	nodeIDbytes := make([]byte, binary.MaxVarintLen64)
	//	var num int
	//	if viperNodeID == 0 {
	//		num = binary.PutUvarint(nodeIDbytes, uint64(serverIndex))
	//	} else {
	//		num = binary.PutUvarint(nodeIDbytes, viperNodeID)
	//	}
	//globals.ID = new(id.Node).SetBytes(nodeIDbytes[:num])

	// Set skipReg from config file
	//globals.SkipRegServer = viper.GetBool("skipReg")

	//	certPath := viper.GetString("certPath")
	//	keyPath := viper.GetString("keyPath")
	//	gatewayCertPath := viper.GetString("gatewayCertPath")
	// Set the certPaths explicitly to avoid data races
	//connect.ServerCertPath = certPath
	//connect.GatewayCertPath = gatewayCertPath
	// Kick off Comms server
	//go node.StartServer(localServer, io.NewServerImplementation(),
	//  certPath, keyPath)

	// TODO Replace these concepts with a better system
	//globals.IsLastNode = serverIndex == len(io.Servers)-1
	//io.NextServer = io.Servers[(serverIndex+1)%len(io.Servers)]

	// Block until we can reach every server
	//io.VerifyServersOnline()

	//globals.RoundRecycle = make(chan *globals.Round, PRECOMP_BUFFER_SIZE)

	// Run as many as half the number of nodes times the number of
	// passthroughs (which is 4).
	//numPrecompSimultaneous = int((uint64(len(io.Servers)) * 4) / 2)
	//if globals.IsLastNode {
	//	realtimeSignal := &sync.Cond{L: &sync.Mutex{}}
	//	io.RoundCh = make(chan *string, PRECOMP_BUFFER_SIZE)
	//	io.MessageCh = make(chan *realtime.Slot, messageBufferSize)
	//	// Last Node handles when realtime and precomp get run
	//	go RunRealTime(batchSize, io.MessageCh, io.RoundCh, realtimeSignal)
	//	go RunPrecomputation(io.RoundCh, realtimeSignal)
	//}

	// Main loop
	//run()
}
