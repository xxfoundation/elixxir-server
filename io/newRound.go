package io

import (
	jww "github.com/spf13/jwalterweatherman"
	pb "gitlab.com/privategrity/comms/mixmessages"
	"gitlab.com/privategrity/comms/mixserver/message"
	"gitlab.com/privategrity/server/cryptops"
	"gitlab.com/privategrity/server/cryptops/precomputation"
	"gitlab.com/privategrity/server/cryptops/realtime"
	"gitlab.com/privategrity/server/globals"
	"gitlab.com/privategrity/server/services"
	"time"
)

// Comms method for kicking off a new round in CMIX
func (s ServerImpl) NewRound() {
	roundId := "TEST"
	batchSize := uint64(5)
	// Create a new Round
	round := globals.NewRound(batchSize)
	// Add round to the GlobalRoundMap
	globals.GlobalRoundMap.AddRound(roundId, round)
	// Initialize the LastNode struct for the round
	if IsLastNode { // TODO better last node system
		globals.InitLastNode(round)
	}

	// Create the controller for PrecompShare
	precompShareController := services.DispatchCryptop(globals.Grp,
		precomputation.Share{}, nil, nil, round)
	// Add the inChannel from the controller to round
	round.AddChannel(globals.PRECOMP_SHARE, precompShareController.InChannel)
	// Kick off PrecompShare Transmission Handler
	services.BatchTransmissionDispatch(roundId, batchSize,
		precompShareController.OutChannel, PrecompShareHandler{})

	// Create the controller for PrecompDecrypt
	precompDecryptController := services.DispatchCryptop(globals.Grp,
		precomputation.Decrypt{}, nil, nil, round)
	// Add the InChannel from the controller to round
	round.AddChannel(globals.PRECOMP_DECRYPT, precompDecryptController.InChannel)
	// Kick off PrecompDecrypt Transmission Handler
	services.BatchTransmissionDispatch(roundId, batchSize,
		precompDecryptController.OutChannel, PrecompDecryptHandler{})

	// Create the controller for PrecompEncrypt
	precompEncryptController := services.DispatchCryptop(globals.Grp,
		precomputation.Encrypt{}, nil, nil, round)
	// Add the InChannel from the controller to round
	round.AddChannel(globals.PRECOMP_ENCRYPT, precompEncryptController.InChannel)
	// Kick off PrecompEncrypt Transmission Handler
	services.BatchTransmissionDispatch(roundId, batchSize,
		precompEncryptController.OutChannel, PrecompEncryptHandler{})

	// Create the controller for PrecompReveal
	precompRevealController := services.DispatchCryptop(globals.Grp,
		precomputation.Reveal{}, nil, nil, round)
	// Add the InChannel from the controller to round
	round.AddChannel(globals.PRECOMP_REVEAL, precompRevealController.InChannel)
	// Kick off PrecompReveal Transmission Handler
	services.BatchTransmissionDispatch(roundId, batchSize,
		precompRevealController.OutChannel, PrecompRevealHandler{})

	// Create the dispatch controller for PrecompPermute
	precompPermuteController := services.DispatchCryptop(globals.Grp,
		precomputation.Permute{}, nil, nil, round)
	// Hook up the dispatcher's input to the round
	round.AddChannel(globals.PRECOMP_PERMUTE,
		precompPermuteController.InChannel)
	// Create the message reorganizer for PrecompPermute
	precompPermuteReorganizer := services.NewSlotReorganizer(
		precompPermuteController.OutChannel, nil, batchSize)
	// Kick off PrecompPermute Transmission Handler
	services.BatchTransmissionDispatch(roundId, batchSize,
		precompPermuteReorganizer.OutChannel,
		PrecompPermuteHandler{})

	// Create the controller for Keygen
	keygenController := services.DispatchCryptop(globals.Grp,
		cryptops.GenerateClientKey{}, nil, nil, round)

	// Create the controller for RealtimeDecrypt
	realtimeDecryptController := services.DispatchCryptop(globals.Grp,
		realtime.Decrypt{}, keygenController.OutChannel, nil, round)
	// Add the InChannel from the keygen controller to round
	round.AddChannel(globals.REAL_DECRYPT, keygenController.InChannel)
	// Kick off RealtimeDecrypt Transmission Handler
	services.BatchTransmissionDispatch(roundId, batchSize,
		realtimeDecryptController.OutChannel, RealtimeDecryptHandler{})

	// Create the controller for RealtimeEncrypt
	realtimeEncryptController := services.DispatchCryptop(globals.Grp,
		realtime.Encrypt{}, keygenController.OutChannel, nil, round)
	// Add the InChannel from the keygen controller to round
	round.AddChannel(globals.REAL_ENCRYPT, keygenController.InChannel)
	// Kick off RealtimeEncrypt Transmission Handler
	services.BatchTransmissionDispatch(roundId, batchSize,
		realtimeEncryptController.OutChannel, RealtimeEncryptHandler{})

	// Create the dispatch controller for RealtimePermute
	realtimePermuteController := services.DispatchCryptop(globals.Grp,
		realtime.Permute{}, nil, nil, round)
	// Hook up the dispatcher's input to the round
	round.AddChannel(globals.REAL_PERMUTE,
		realtimePermuteController.InChannel)
	// Create the message reorganizer for RealtimePermute
	realtimePermuteReorganizer := services.NewSlotReorganizer(
		realtimePermuteController.OutChannel, nil, batchSize)
	// Kick off RealtimePermute Transmission Handler
	services.BatchTransmissionDispatch(roundId, batchSize,
		realtimePermuteReorganizer.OutChannel,
		RealtimePermuteHandler{})

	if IsLastNode { // TODO better last node system
		// Create the controller for RealtimeIdentify
		realtimeIdentifyController := services.DispatchCryptop(globals.Grp,
			realtime.Identify{}, nil, nil, round)
		// Add the InChannel from the controller to round
		round.AddChannel(globals.REAL_IDENTIFY, realtimeIdentifyController.InChannel)
		// Kick off RealtimeIdentify Transmission Handler
		services.BatchTransmissionDispatch(roundId, batchSize,
			realtimeIdentifyController.OutChannel, RealtimeIdentifyHandler{})
	}

	// Create the dispatch controller for PrecompGeneration
	precompGenerationController := services.DispatchCryptop(globals.Grp,
		precomputation.Generation{}, nil, nil, round)
	// Run PrecompGeneration for the entire batch
	for j := uint64(0); j < batchSize; j++ {
		genMsg := services.Slot(&precomputation.SlotGeneration{Slot: j})
		precompGenerationController.InChannel <- &genMsg
		_ = <-precompGenerationController.OutChannel
	}
}

// Blocks until all given servers begin a new round
func BeginNewRound(servers []string) {
	for i := 0; i < len(servers); {
		jww.INFO.Printf("Sending NewRound message to %s...", servers[i])
		_, err := message.SendNewRound(servers[i], &pb.InitRound{})
		if err != nil {
			jww.ERROR.Printf("%v: Server %s failed to begin new round!", i, servers[i])
			time.Sleep(250 * time.Millisecond)
		} else {
			jww.INFO.Printf("%v: Server %s began new round!", i, servers[i])
			i++
		}
	}
}
