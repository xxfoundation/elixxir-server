// Package node contains the initialization and main loop of a cMix server.
package node

import (
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"
	"strconv"

	"gitlab.com/privategrity/comms/mixserver"
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/server/globals"
	"gitlab.com/privategrity/server/io"
)

// StartServer reads configuration options and starts the cMix server
func StartServer(serverIndex int) {
	viper.Debug()
	jww.INFO.Printf("Log Filename: %v\n", viper.GetString("logPath"))
	jww.INFO.Printf("Config Filename: %v\n\n", viper.ConfigFileUsed())

	// Get all servers
	servers := getServers()

	// TODO Generate globals.Grp somewhere intelligent
	rng := cyclic.NewRandom(cyclic.NewInt(0), cyclic.NewInt(1000))
	grp := cyclic.NewGroup(cyclic.NewInt(101), cyclic.NewInt(5), cyclic.NewInt(4),
		rng)
	globals.Grp = &grp

	// Start mix servers on localServer
	localServer := servers[serverIndex]
	jww.INFO.Printf("Starting server on %v\n", localServer)
	// Initialize GlobalRoundMap
	globals.GlobalRoundMap = globals.NewRoundMap()
	// Kick off Comms server
	go mixserver.StartServer(localServer, io.ServerImpl{Rounds: &globals.GlobalRoundMap})

	// Block until we can reach every server
	io.VerifyServersOnline(servers)

	// TODO Replace these booleans with a better system
	if serverIndex == len(servers)-1 {
		// Next leap will be first server
		io.NextServer = servers[0]
		// We are the last node
		io.IsLastNode = true
		// Begin the round on all nodes
		io.BeginNewRound(servers)
	} else {
		// Not last server, next leap will be next server
		io.NextServer = servers[serverIndex+1]
		io.IsLastNode = false
	}

	// Main loop
	run()
}

// Main server loop
func run() {
	// Blocks forever as a keepalive
	select {}
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
		// Assemble full server address
		servers[i] = "localhost:" + servers[i]
	}
	return servers
}
