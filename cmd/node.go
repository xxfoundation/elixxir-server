////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Package node contains the initialization and main loop of a cMix server.
package cmd

import (
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/csprng"
	"gitlab.com/elixxir/crypto/signature"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/globals"
	"gitlab.com/elixxir/server/io"
	"gitlab.com/elixxir/server/node"
	"gitlab.com/elixxir/server/server"
	"gitlab.com/elixxir/server/server/phase"
	"time"

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

	//Check that there is a gateway
	if len(params.Gateways) < 1 {
		// No gateways in config file or passed via command line
		jww.FATAL.Panicf("Error: No gateways specified! Add to" +
			" configuration file!")
		return
	}

	// Initialize the backend
	dbAddress := params.Database.Addresses[params.Index]

	userDatabase := globals.NewUserRegistry(
		params.Database.Username,
		params.Database.Password,
		params.Database.Name,
		dbAddress,
	)

	//Build DSA key
	rng := csprng.NewSystemRNG()
	grp := params.Groups.CMix
	dsaParams := signature.CustomDSAParams(grp.GetP(), grp.GetQ(), grp.GetG())
	privKey := dsaParams.PrivateKeyGen(rng)
	pubKey := privKey.PublicKeyGen()

	//TODO: store DSA key for NDF

	// Create instance
	instance := server.CreateServerInstance(params, userDatabase, pubKey, privKey)

	// initialize the network
	instance.InitNetwork(node.NewImplementation)

	//FIXME: check that all other nodes are online

	//Begin the resource queue
	instance.Run()

	//Start runners for first node
	if instance.IsFirstNode() {
		instance.InitFirstNode()

		batchSize := params.Batch

		starter := func(instance *server.Instance, rid id.Round) error {
			newBatch := &mixmessages.Batch{
				Slots:    make([]*mixmessages.Slot, batchSize),
				ForPhase: int32(phase.PrecompGeneration),
				Round: &mixmessages.RoundInfo{
					ID: uint64(rid),
				},
			}
			for i := 0; i < int(batchSize); i++ {
				newBatch.Slots[i] = &mixmessages.Slot{}
			}

			node.ReceivePostPhase(newBatch, instance)
			return nil
		}

		instance.RunFirstNode(instance, 10*time.Second,
			io.TransmitCreateNewRound, starter)
	}
}
