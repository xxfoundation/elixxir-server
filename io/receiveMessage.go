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

// Serves as the batch queue
// TODO better batch logic, we should convert this to a queue or channel
var msgCounter uint64 = 0
var msgQueue = make([]*pb.CmixMessage, globals.BatchSize)

// Reception handler for ReceiveMessageFromClient
func (s ServerImpl) ReceiveMessageFromClient(msg *pb.CmixMessage) {
	jww.DEBUG.Printf("Received message from client: %v...", msg)

	// Append the message to the batch queue
	msgQueue[msgCounter] = msg
	msgCounter += 1

	// Once the batch is filled
	if msgCounter == globals.BatchSize {
		roundId := globals.GetNextWaitingRoundID()
		jww.DEBUG.Printf("Beginning round %s...", roundId)
		// Pass the batch queue into Realtime
		StartRealtime(msgQueue, roundId, globals.BatchSize)

		// Reset the batch queue
		msgCounter = 0
		msgQueue = make([]*pb.CmixMessage, globals.BatchSize)
		// Begin a new round and start precomputation
		BeginNewRound(Servers)
	}
}

// Begin Realtime once Precomputation is finished
func StartRealtime(messages []*pb.CmixMessage, roundId string, batchSize uint64) {
	inputSlots := make([]*services.Slot, batchSize)
	for i := uint64(0); i < batchSize; i++ {
		inputMsg := services.Slot(&realtime.SlotDecryptOut{
			Slot:                 i,
			SenderID:             1,
			EncryptedMessage:     cyclic.NewIntFromBytes(messages[i].MessagePayload),
			EncryptedRecipientID: cyclic.NewIntFromBytes(messages[i].RecipientID),
		})
		inputSlots[i] = &inputMsg
	}

	jww.INFO.Println("Beginning RealtimeDecrypt Phase...")
	kickoffDecryptHandler(roundId, batchSize, inputSlots)
}
