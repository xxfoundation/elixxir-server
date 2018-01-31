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
	//"gitlab.com/privategrity/comms/mixserver/message"
	//"gitlab.com/privategrity/crypto/cyclic"
	//"gitlab.com/privategrity/server/cryptops/precomputation"
	"gitlab.com/privategrity/server/node"
	//"gitlab.com/privategrity/server/services"
	"gitlab.com/privategrity/server/services"
)

// TODO move or remove this probs
var nextServer string

// Blank struct implementing mixserver.ServerHandler interface TODO put this somewhere lol
type ServerImpl struct {
	// Pointer to the global map of RoundID -> Rounds
	rounds *node.RoundMap
}

// Get the respective channel for the given roundId and chanId combination
func (s ServerImpl) GetChannel(roundId string, chanId uint8) chan<- *services.Slot {
	return s.rounds.GetRound(roundId).GetChannel(chanId)
}

// Run is the main loop for the cMix server
func run() {
	// TODO literally anything
	time.Sleep(5 * time.Second)
}

// TODO
func (h ServerImpl) PrecompDecrypt(input *pb.PrecompDecryptMessage) {

	// Convert input message to equivalent SlotDecrypt
	//slotDecrypt := &precomputation.SlotDecrypt{
	//	Slot:                         input.Slot,
	//	EncryptedMessageKeys:         cyclic.NewIntFromBytes(input.EncryptedMessageKeys),
	//	EncryptedRecipientIDKeys:     cyclic.NewIntFromBytes(input.EncryptedRecipientIDKeys),
	//	PartialMessageCypherText:     cyclic.NewIntFromBytes(input.PartialMessageCypherText),
	//	PartialRecipientIDCypherText: cyclic.NewIntFromBytes(input.PartialRecipientIDCypherText),
	//}
	// Type assert SlotDecrypt to Slot
	//var slot services.Slot = slotDecrypt

	// Pass slot as input to Decrypt

	// Get output from Decrypt
	//output := <-dcPrecompDecrypt.OutChannel
	// Type assert Slot to SlotDecrypt
	//out := (*output).(*precomputation.SlotDecrypt)

	//_, err = c.PrecompDecrypt(ctx, &pb.PrecompDecryptMessage{
	//	Slot:                         out.Slot,
	//	EncryptedMessageKeys:         out.EncryptedMessageKeys.Bytes(),
	//	EncryptedRecipientIDKeys:     out.EncryptedRecipientIDKeys.Bytes(),
	//	PartialMessageCypherText:     out.PartialMessageCypherText.Bytes(),
	//	PartialRecipientIDCypherText: out.PartialRecipientIDCypherText.Bytes(),
	//})
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
	// Initialize GlobalRoundMap
	node.GlobalRoundMap = node.NewRoundMap()
	// Kick off Comms server
	go mixserver.StartServer(localServer, ServerImpl{rounds: &node.GlobalRoundMap})

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
