// Package node contains the initialization and main loop of a cMix server.
package server

import (
	"strconv"
	"time"

	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"
	"golang.org/x/net/context"
	"google.golang.org/grpc"

	pb "gitlab.com/privategrity/comms/mixmessages"
	"gitlab.com/privategrity/comms/mixserver"
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/server/cryptops/precomputation"
	"gitlab.com/privategrity/server/node"
	"gitlab.com/privategrity/server/services"
)

var nextServer string

// Run is the main loop for the cMix server
func run() {

}

func precompDecrypt(input pb.PrecompDecryptMessage) {
	// Create a new round TODO remove I guess
	round := node.NewRound(5)

	// Create Dispatcher for Decrypt
	dcPrecompDecrypt := services.DispatchCryptop(node.Grp, precomputation.Decrypt{}, nil, nil, round)

	// Convert input message to equivalent SlotDecrypt
	slotDecrypt := &precomputation.SlotDecrypt{
		EncryptedMessageKeys:         cyclic.NewIntFromBytes(input.EncryptedMessageKeys),
		EncryptedRecipientIDKeys:     cyclic.NewIntFromBytes(input.EncryptedRecipientIDKeys),
		PartialMessageCypherText:     cyclic.NewIntFromBytes(input.PartialMessageCypherText),
		PartialRecipientIDCypherText: cyclic.NewIntFromBytes(input.PartialRecipientIDCypherText),
	}
	// Type assert SlotDecrypt to Slot
	var slot services.Slot = slotDecrypt

	// Pass slot as input to Decrypt
	dcPrecompDecrypt.InChannel <- &slot

	// Get output from Decrypt
	output := <-dcPrecompDecrypt.OutChannel
	// Type assert Slot to SlotDecrypt
	out := (*output).(*precomputation.SlotDecrypt)

	// Attempt to connect to nextServer
	conn, err := grpc.Dial(nextServer, grpc.WithInsecure())
	// Check for an error
	if err != nil {
		jww.ERROR.Printf("Failed to connect to server at %v\n", nextServer)
	}
	c := pb.NewMixMessageServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	// Send the PrecompDecrypt message using the Decrypt output
	_, err = c.PrecompDecrypt(ctx, &pb.PrecompDecryptMessage{
		EncryptedMessageKeys:         out.EncryptedMessageKeys.Bytes(),
		EncryptedRecipientIDKeys:     out.EncryptedRecipientIDKeys.Bytes(),
		PartialMessageCypherText:     out.PartialMessageCypherText.Bytes(),
		PartialRecipientIDCypherText: out.PartialRecipientIDCypherText.Bytes(),
	})
	// Make sure there are no errors with sending the message
	if err != nil {
		jww.ERROR.Printf("PrecompDecrypt: Error received: %s", err)
	}
}

// Checks to see if the given servers are online TODO something with this
func verifyServersOnline(servers []string) {
	for i := range servers {
		// Connect to server with gRPC
		jww.INFO.Printf("Connecting to server on port %v\n", servers[i])
		addr := "localhost:" + servers[i]
		conn, err := grpc.Dial(addr, grpc.WithInsecure())
		if err != nil {
			jww.ERROR.Printf("Failed to connect to server at %v\n", addr)
		}
		time.Sleep(time.Millisecond * 500)

		c := pb.NewMixMessageServiceClient(conn)
		// Send AskOnline Request and check that we get an AskOnlineAck back
		ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
		_, err = c.AskOnline(ctx, &pb.Ping{})
		if err != nil {
			jww.ERROR.Printf("AskOnline: Error received: %s", err)
		} else {
			jww.INFO.Printf("AskOnline: %v is online!", servers[i])
		}
		cancel()
		conn.Close()
	}
}

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
	nextServer = "localhost:" + servers[nextIndex]
	localServer := "localhost:" + servers[serverIndex]

	// Start mix servers on localServer
	jww.INFO.Printf("Starting server on %v\n", localServer)
	go mixserver.StartServer(localServer)

	// Main loop
	run()
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
