// Package node contains the initialization and main loop of a cMix server.
package node

import (
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"
	"strconv"

	"gitlab.com/privategrity/comms/mixserver"
	"gitlab.com/privategrity/server/cryptops/precomputation"
	"gitlab.com/privategrity/server/io"
	"gitlab.com/privategrity/server/services"
	"gitlab.com/privategrity/server/globals"
)

// StartServer reads configuration options and starts the cMix server
func StartServer(serverIndex int) {
	viper.Debug()
	jww.INFO.Printf("Log Filename: %v\n", viper.GetString("logPath"))
	jww.INFO.Printf("Config Filename: %v\n\n", viper.ConfigFileUsed())

	// Get all servers
	servers := getServers()
	// Determine what the next server leap is
	nextIndex := serverIndex + 1
	if serverIndex == len(servers)-1 {
		nextIndex = 0
	}
	io.NextServer = "localhost:" + servers[nextIndex]
	localServer := "localhost:" + servers[serverIndex]

	// Start mix servers on localServer
	jww.INFO.Printf("Starting server on %v\n", localServer)
	// Initialize GlobalRoundMap
	globals.GlobalRoundMap = globals.NewRoundMap()
	// Kick off Comms server
	go mixserver.StartServer(localServer, io.ServerImpl{Rounds: &globals.GlobalRoundMap})
	// Kick off a Round TODO Better parameters
	NewRound("Test", 5)
}

// getServers pulls a string slice of server ports from the config file and
// verifies that the ports are valid.
func getServers() []string {
	servers := viper.GetStringSlice("servers")
	if servers == nil {
		jww.ERROR.Println("No servers listed in config file.")
	}
	for i := range servers {
		temp, err := strconv.Atoi(servers[i])
		// catch non-int ports
		if err != nil {
			jww.ERROR.Println("Non-integer server ports in config file")
		}
		// Catch invalid ports
		if temp > 65535 || temp < 0 {
			jww.ERROR.Printf("Port %v listed in the config file is not a "+
				"valid port\n", temp)
		}
		// Catch reserved ports
		if temp < 1024 {
			jww.WARN.Printf("Port %v is a reserved port, superuser privilege"+
				" may be required.\n", temp)
		}
	}
	return servers
}

// Kicks off a new round in CMIX
func NewRound(roundId string, batchSize uint64) {
	// Create a new Round
	round := globals.NewRound(batchSize)
	// Add round to the GlobalRoundMap
	globals.GlobalRoundMap.AddRound(roundId, round)

	// Create the controller for PrecompDecrypt
	precompDecryptCntrlr := services.DispatchCryptop(globals.Grp,
		precomputation.Decrypt{}, nil, nil, round)
	// Add the InChannel from the controller to round
	round.AddChannel(globals.PRECOMP_DECRYPT, precompDecryptCntrlr.InChannel)
	// Kick off PrecompDecrypt  Transmission Handler
	services.BatchTransmissionDispatch(roundId, batchSize,
		precompDecryptCntrlr.OutChannel, io.PrecompDecryptHandler{})
}
