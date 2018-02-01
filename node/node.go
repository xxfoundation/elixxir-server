// Package node contains the initialization and main loop of a cMix server.
package node

import (
	"strconv"
	"time"

	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"

	pb "gitlab.com/privategrity/comms/mixmessages"
	"gitlab.com/privategrity/comms/mixserver"
	"gitlab.com/privategrity/comms/mixserver/message"
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/server/cryptops/precomputation"
	"gitlab.com/privategrity/server/globals"
	"gitlab.com/privategrity/server/services"
)

// Address of the subsequent server in the config file
// TODO move or remove this probably
var nextServer string

// Blank struct implementing mixserver.ServerHandler interface
type ServerImpl struct {
	// Pointer to the global map of RoundID -> Rounds
	rounds *globals.RoundMap
}

// Get the respective channel for the given roundId and chanId combination
func (s ServerImpl) GetChannel(roundId string, chanId globals.Phase) chan<- *services.Slot {
	return s.rounds.GetRound(roundId).GetChannel(chanId)
}

// Run is the main loop for the cMix server
func run() {
	// TODO literally anything
	time.Sleep(5 * time.Second)
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
	// Kick off PrecompDecryptTransmissionHandler
	go precompDecryptTransmissionHandler(roundId, batchSize, precompDecryptCntrlr.OutChannel)
}

// ReceptionHandler for PrecompDecryptMessages
func (s ServerImpl) PrecompDecrypt(input *pb.PrecompDecryptMessage) {
	// Iterate through the Slots in the PrecompDecryptMessage
	for i := 0; i < len(input.Slots); i++ {
		// Convert input message to equivalent SlotDecrypt
		in := input.Slots[i]
		slotDecrypt := &precomputation.SlotDecrypt{
			Slot:                         in.Slot,
			EncryptedMessageKeys:         cyclic.NewIntFromBytes(in.EncryptedMessageKeys),
			EncryptedRecipientIDKeys:     cyclic.NewIntFromBytes(in.EncryptedRecipientIDKeys),
			PartialMessageCypherText:     cyclic.NewIntFromBytes(in.PartialMessageCypherText),
			PartialRecipientIDCypherText: cyclic.NewIntFromBytes(in.PartialRecipientIDCypherText),
		}
		// Type assert SlotDecrypt to Slot
		var slot services.Slot = slotDecrypt

		// Pass slot as input to Decrypt's channel
		s.GetChannel(input.RoundID, globals.PRECOMP_DECRYPT) <- &slot
	}
}

// TransmissionHandler for PrecompDecryptMessages
func precompDecryptTransmissionHandler(roundId string, batchSize uint64, outCh chan *services.Slot) {
	// Get the round BatchSize
	bs := globals.GlobalRoundMap.GetRound(roundId).BatchSize

	// Create the PrecompDecryptMessage
	msg := &pb.PrecompDecryptMessage{
		RoundID: roundId,
		Slots:   make([]*pb.PrecompDecryptSlot, bs),
	}

	// Iterate over the output channel
	for i := uint64(0); i < bs; i++ {
		// Get output from Decrypt TODO
		output := <-outCh
		// Type assert Slot to SlotDecrypt
		out := (*output).(*precomputation.SlotDecrypt)
		// Convert to PrecompDecryptSlot
		msgSlot := &pb.PrecompDecryptSlot{
			Slot:                         out.Slot,
			EncryptedMessageKeys:         out.EncryptedMessageKeys.Bytes(),
			EncryptedRecipientIDKeys:     out.EncryptedRecipientIDKeys.Bytes(),
			PartialMessageCypherText:     out.PartialMessageCypherText.Bytes(),
			PartialRecipientIDCypherText: out.PartialRecipientIDCypherText.Bytes(),
		}

		// Append the PrecompDecryptSlot to the PrecompDecryptMessage
		msg.Slots = append(msg.Slots, msgSlot)
	}
	// Send the completed PrecompDecryptMessage
	message.SendPrecompDecrypt(nextServer, msg)
}

// Checks to see if the given servers are online
func verifyServersOnline(servers []string) {
	for i := range servers {
		_, err := message.SendAskOnline(servers[i], &pb.Ping{})
		if err != nil {
			jww.ERROR.Println("Server %s failed to respond!", servers[i])
		}
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
	globals.GlobalRoundMap = globals.NewRoundMap()
	// Kick off Comms server
	go mixserver.StartServer(localServer, ServerImpl{rounds: &globals.GlobalRoundMap})
	// Kick off a Round TODO
	NewRound("Test", 5)

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
