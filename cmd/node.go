////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Package node contains the initialization and main loop of a cMix server.
package cmd

import (
	"encoding/json"
	"gitlab.com/elixxir/crypto/csprng"
	"gitlab.com/elixxir/crypto/large"
	"gitlab.com/elixxir/crypto/signature"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/globals"
	"gitlab.com/elixxir/server/io"
	"gitlab.com/elixxir/server/node"
	"gitlab.com/elixxir/server/server"
	"io/ioutil"
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
)

// StartServer reads configuration options and starts the cMix server
func StartServer(vip *viper.Viper) {
	vip.Debug()

	jww.INFO.Printf("Log Filename: %v\n", vip.GetString("node.paths.log"))
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

	jww.INFO.Printf("Loaded params: %v", params)

	//Check that there is a gateway
	if len(params.Gateways.Addresses) < 1 {
		// No gateways in config file or passed via command line
		jww.FATAL.Panicf("Error: No gateways specified! Add to" +
			" configuration file!")
		return
	}

	// Initialize the backend
	dbAddress := params.Database.Addresses[params.Index]
	grp := params.Groups.GetCMix()

	// Initialize the global group
	globals.SetGroup(grp)

	//Initialize the user database
	userDatabase := globals.NewUserRegistry(
		params.Database.Username,
		params.Database.Password,
		params.Database.Name,
		dbAddress,
	)

	//Add a dummy user for gateway
	dummy := userDatabase.NewUser(grp)
	dummy.ID = id.MakeDummyUserID()
	dummy.BaseKey = grp.NewIntFromBytes((*dummy.ID)[:])
	userDatabase.UpsertUser(dummy)

	//Build DSA key

	var privKey *signature.DSAPrivateKey
	var pubKey *signature.DSAPublicKey

	rng := csprng.NewSystemRNG()
	dsaParams := signature.CustomDSAParams(grp.GetP(), grp.GetQ(), grp.GetG())

	if dsaKeyPairPath == "" {
		privKey = dsaParams.PrivateKeyGen(rng)
		pubKey = privKey.PublicKeyGen()
	} else {
		dsaKeyBytes, err := ioutil.ReadFile(dsaKeyPairPath)

		if err != nil {
			jww.FATAL.Panicf("Could not read dsa keys file: %v", err)
		}

		dsaKeys := DSAKeysJson{}

		err = json.Unmarshal(dsaKeyBytes, &dsaKeys)

		if err != nil {
			jww.FATAL.Panicf("Could not unmarshal dsa keys file: %v", err)
		}

		dsaPrivInt := large.NewIntFromString(dsaKeys.PrivateKeyHex, 16)
		dsaPubInt := large.NewIntFromString(dsaKeys.PublicKeyHex, 16)

		pubKey = signature.ReconstructPublicKey(dsaParams, dsaPubInt)
		privKey = signature.ReconstructPrivateKey(pubKey, dsaPrivInt)
	}

	//TODO: store DSA key for NDF

	// Create instance
	instance := server.CreateServerInstance(params, userDatabase, pubKey, privKey)

	if instance.IsFirstNode() {
		instance.InitFirstNode()
	}
	if instance.IsLastNode() {
		instance.InitLastNode()
	}

	// initialize the network
	instance.InitNetwork(node.NewImplementation)

	// Check that all other nodes are online
	io.VerifyServersOnline(instance.GetNetwork(), instance.GetTopology())

	//Begin the resource queue
	instance.Run()

	//Start runners for first node
	if instance.IsFirstNode() {
		instance.RunFirstNode(instance, roundBufferTimeout*time.Second,
			io.TransmitCreateNewRound, node.MakeStarter(params.Batch))
	}

}

type DSAKeysJson struct {
	PrivateKeyHex string
	PublicKeyHex  string
}
