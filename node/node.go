// Package node contains the initialization and main loop of a cMix server.
package node

import (
	"strconv"
	"time"

	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"
	"golang.org/x/net/context"
	"google.golang.org/grpc"

	pb "gitlab.com/privategrity/comms/mixmessages"
	"gitlab.com/privategrity/comms/mixserver"
)

// Run is the main loop for the cMix server
func Run(servers []string) {
	for i := range servers {
		// Start mixservers on localhost port
		jww.INFO.Printf("Starting server on port %v\n", servers[i])
		go mixserver.StartServer("localhost:" + servers[i])

	}
	for i := range servers {
		// Connect to server with gRPC
		jww.INFO.Printf("Connecting to server on port %v\n", servers[i])
		addr := "localhost:" + servers[i]
		conn, err := grpc.Dial(addr, grpc.WithInsecure(),
			grpc.WithTimeout(time.Second))
		if err != nil {
			jww.ERROR.Printf("Failed to connect to server at %v\n", addr)
		}
		defer conn.Close()
		time.Sleep(time.Millisecond * 500)
		c := pb.NewMixMessageServiceClient(conn)

		// Send AskOnline Request and check that we get an AskOnlineAck back
		ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
		response, err := c.AskOnline(ctx, &pb.AskOnlineRequest{})
		if err != nil {
			jww.ERROR.Printf("AskOnline: Error received: %s", err)
		}
		if !response.IsOnline {
			jww.ERROR.Printf("AskOnline: Failed to get an online confirmation!")
		} else {
			jww.INFO.Printf("AskOnline: %v is online!", addr)
		}
		defer cancel()
	}
	time.Sleep(time.Millisecond * 1000)

}

// StartServer reads configuration options and starts the cMix server
func StartServer() {
	viper.Debug()
	jww.INFO.Printf("Log Filename: %v\n", viper.GetString("logPath"))
	jww.INFO.Printf("Config Filename: %v\n\n", viper.ConfigFileUsed())

	Run(getServers())
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
