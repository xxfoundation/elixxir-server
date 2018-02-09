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
	io.Servers = getServers()

	// TODO Generate globals.Grp somewhere intelligent
	rng := cyclic.NewRandom(cyclic.NewInt(0), cyclic.NewInt(1000))
	grp := cyclic.NewGroup(cyclic.NewInt(101), cyclic.NewInt(5), cyclic.NewInt(4),
		rng)
	globals.Grp = &grp

	// Start mix servers on localServer
	localServer := io.Servers[serverIndex]
	jww.INFO.Printf("Starting server on %v\n", localServer)
	// Initialize GlobalRoundMap
	globals.GlobalRoundMap = globals.NewRoundMap()
	// Kick off Comms server
	go mixserver.StartServer(localServer, io.ServerImpl{
		Rounds: &globals.GlobalRoundMap,
	})

	// TODO Replace these concepts with a better system
	io.IsLastNode = serverIndex == len(io.Servers)-1
	io.NextServer = io.Servers[(serverIndex+1)%len(io.Servers)]

	// Block until we can reach every server
	io.VerifyServersOnline(io.Servers)

	if io.IsLastNode {
		// Begin the round on all nodes
		io.BeginNewRound(io.Servers)
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
