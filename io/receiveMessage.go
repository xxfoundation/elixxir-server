package io

import (
	jww "github.com/spf13/jwalterweatherman"
	pb "gitlab.com/privategrity/comms/mixmessages"
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/server/cryptops/realtime"
	"gitlab.com/privategrity/server/globals"
	"gitlab.com/privategrity/server/services"
)

type ReceiveMessageHandler struct{}

// Reception handler for ReceiveMessageFromClient
func (s ServerImpl) ReceiveMessageFromClient(msg *pb.CmixMessage) {
	jww.DEBUG.Printf("Received message from client: %v...", msg)

	// Start precomputation for the next message we receive
	BeginNewRound(Servers)

	roundId := globals.GetNextWaitingRoundID()

	jww.DEBUG.Printf("roundID: %s\n", roundId)

	StartRealtime(msg, roundId, 1)
}

// Begin Realtime once Precomputation is finished
func StartRealtime(msg *pb.CmixMessage, roundId string, batchSize uint64) {

	inputMsg := services.Slot(&realtime.SlotDecryptOut{
		Slot:                 0,
		SenderID:             1,
		EncryptedMessage:     cyclic.NewIntFromBytes(msg.MessagePayload),
		EncryptedRecipientID: cyclic.NewIntFromBytes(msg.RecipientID),
	})

	jww.INFO.Println("Beginning RealtimeDecrypt Phase...")
	kickoffDecryptHandler(roundId, batchSize, []*services.Slot{&inputMsg})
}
