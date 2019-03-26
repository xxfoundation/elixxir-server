////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package io

import (
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/server/cryptops"
	"gitlab.com/elixxir/server/cryptops/precomputation"
	"gitlab.com/elixxir/server/cryptops/realtime"
	"gitlab.com/elixxir/server/globals"
	"gitlab.com/elixxir/server/services"

	jww "github.com/spf13/jwalterweatherman"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/primitives/id"
	"time"
)

// Comms method for kicking off a new round in CMIX
func NewRound(clusterRoundID string) {
	startTime := time.Now()
	jww.INFO.Printf("Starting NewRound(RoundId: %s) at %s",
		clusterRoundID,
		startTime.Format(time.RFC3339))

	batchSize := globals.BatchSize
	roundId := globals.GetNextRoundID()
	jww.INFO.Printf("Received Cluster Round ID: %s\n", clusterRoundID)
	if roundId != clusterRoundID {
		jww.FATAL.Printf("round id %s does not match passed round id %s",
			roundId, clusterRoundID)
		panic("Passed round identifier does not match generated identifier!")
	}

	// Create a new Round
	round := globals.NewRound(batchSize, globals.GetGroup())

	// Add round to the GlobalRoundMap
	globals.GlobalRoundMap.AddRound(roundId, round)

	// Timeout this round on this node for precomputation after 10 minutes to
	// prevent deadlock
	// To test timing out precomputation, switch the commented line
	timeoutPrecomputation(roundId, 10*time.Minute)
	//timeoutPrecomputation(roundId, 200*time.Millisecond)

	globals.GlobalRoundMap.SetPhase(roundId, globals.PRECOMP_GENERATION)

	// Create the controller for PrecompShare
	// Note: Share requires a batchSize of 1
	precompShareController := services.DispatchCryptopSized(globals.GetGroup(),
		precomputation.Share{}, nil, nil, uint64(1), round)
	// Add the inChannel from the controller to round
	round.AddChannel(globals.PRECOMP_SHARE, precompShareController.InChannel)
	// Kick off PrecompShare Transmission Handler
	// Note: Share requires a batchSize of 1
	services.BatchTransmissionDispatch(roundId, uint64(1),
		precompShareController.OutChannel, PrecompShareHandler{})

	// Create the controller for PrecompDecrypt
	precompDecryptController := services.DispatchCryptop(globals.GetGroup(),
		precomputation.Decrypt{}, nil, nil, round)
	// Add the InChannel from the controller to round
	round.AddChannel(globals.PRECOMP_DECRYPT, precompDecryptController.InChannel)
	// Kick off PrecompDecrypt Transmission Handler
	services.BatchTransmissionDispatch(roundId, batchSize,
		precompDecryptController.OutChannel, PrecompDecryptHandler{})

	// Create the controller for PrecompEncrypt
	precompEncryptController := services.DispatchCryptop(globals.GetGroup(),
		precomputation.Encrypt{}, nil, nil, round)
	// Add the InChannel from the controller to round
	round.AddChannel(globals.PRECOMP_ENCRYPT, precompEncryptController.InChannel)
	// Kick off PrecompEncrypt Transmission Handler
	services.BatchTransmissionDispatch(roundId, batchSize,
		precompEncryptController.OutChannel, PrecompEncryptHandler{})

	// Create the controller for PrecompReveal
	precompRevealController := services.DispatchCryptop(globals.GetGroup(),
		precomputation.Reveal{}, nil, nil, round)
	// Add the InChannel from the controller to round
	round.AddChannel(globals.PRECOMP_REVEAL, precompRevealController.InChannel)
	// Kick off PrecompReveal Transmission Handler
	services.BatchTransmissionDispatch(roundId, batchSize,
		precompRevealController.OutChannel, PrecompRevealHandler{})

	// Create the dispatch controller for PrecompPermute
	precompPermuteController := services.DispatchCryptop(globals.GetGroup(),
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

	// Create the reception keygen for RealtimeDecrypt
	receptionKeygenInit := make([]interface{}, 2)
	receptionKeygenInit[0] = round
	receptionKeygenInit[1] = cryptops.TRANSMISSION
	realtimeReceptionKeygen := services.DispatchCryptop(globals.GetGroup(),
		cryptops.GenerateClientKey{}, nil, nil, receptionKeygenInit)
	// Create the controller for RealtimeDecrypt
	realtimeDecryptController := services.DispatchCryptop(globals.GetGroup(),
		realtime.Decrypt{}, realtimeReceptionKeygen.OutChannel, nil, round)
	// Add the InChannel from the keygen controller to round
	round.AddChannel(globals.REAL_DECRYPT, realtimeReceptionKeygen.InChannel)
	// Kick off RealtimeDecrypt Transmission Handler
	services.BatchTransmissionDispatch(roundId, batchSize,
		realtimeDecryptController.OutChannel, RealtimeDecryptHandler{})

	// Create the transmission keygen for RealtimeEncrypt
	transmissionKeygenInit := make([]interface{}, 2)
	transmissionKeygenInit[0] = round
	transmissionKeygenInit[1] = cryptops.RECEPTION
	realtimeTransmissionKeygen := services.DispatchCryptop(globals.GetGroup(),
		cryptops.GenerateClientKey{}, nil, nil, transmissionKeygenInit)
	// Create the controller for RealtimeEncrypt
	realtimeEncryptController := services.DispatchCryptop(globals.GetGroup(),
		realtime.Encrypt{}, realtimeTransmissionKeygen.OutChannel, nil, round)
	// Add the InChannel from the keygen controller to round
	round.AddChannel(globals.REAL_ENCRYPT, realtimeTransmissionKeygen.InChannel)
	// Kick off RealtimeEncrypt Transmission Handler
	services.BatchTransmissionDispatch(roundId, batchSize,
		realtimeEncryptController.OutChannel, RealtimeEncryptHandler{})

	// Create the dispatch controller for RealtimePermute
	realtimePermuteController := services.DispatchCryptop(globals.GetGroup(),
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

	// Create the dispatch controller for PrecompGeneration
	precompGenerationController := services.DispatchCryptop(globals.GetGroup(),
		precomputation.Generation{}, nil, nil, round)

	genstart := time.Now()
	// Run PrecompGeneration for the entire batch
	for j := uint64(0); j < batchSize; j++ {
		genMsg := services.Slot(&precomputation.SlotGeneration{Slot: j})
		precompGenerationController.InChannel <- &genMsg
	}

	for j := uint64(0); j < batchSize; j++ {
		_ = <-precompGenerationController.OutChannel
	}

	gendlta := time.Now().Sub(genstart)
	jww.DEBUG.Printf("Generate took: %v", gendlta)

	globals.GlobalRoundMap.SetPhase(roundId, globals.PRECOMP_SHARE)

	if id.IsLastNode {
		// Create the controller for RealtimeIdentify
		realtimeIdentifyController := services.DispatchCryptop(globals.GetGroup(),
			realtime.Identify{}, nil, nil, round)
		// Add the InChannel from the controller to round
		round.AddChannel(globals.REAL_IDENTIFY,
			realtimeIdentifyController.InChannel)
		//Add Verify on the end of Identify
		realtimeVerifyController := services.DispatchCryptop(globals.GetGroup(),
			realtime.Verify{}, realtimeIdentifyController.OutChannel, nil, round)
		// Kick off RealtimeIdentify Transmission Handler
		services.BatchTransmissionDispatch(roundId, batchSize,
			realtimeVerifyController.OutChannel, RealtimeIdentifyHandler{})

		// Create the controller for RealtimePeel
		realtimePeelController := services.DispatchCryptop(globals.GetGroup(),
			realtime.Peel{}, nil, nil, round)
		// Add the InChannel from the controller to round
		round.AddChannel(globals.REAL_PEEL, realtimePeelController.InChannel)
		// Kick off RealtimePeel Transmission Handler
		services.BatchTransmissionDispatch(roundId, batchSize,
			realtimePeelController.OutChannel, RealtimePeelHandler{})

		// Create the controller for PrecompStrip
		precompStripController := services.DispatchCryptop(globals.GetGroup(),
			precomputation.Strip{}, nil, nil, round)
		// Add the InChannel from the controller to round
		round.AddChannel(globals.PRECOMP_STRIP, precompStripController.InChannel)
		// Kick off PrecompStrip Transmission Handler
		services.BatchTransmissionDispatch(roundId, batchSize,
			precompStripController.OutChannel, PrecompStripHandler{})

		jww.INFO.Println("Beginning PrecompShare Phase...")
		shareMsg := services.Slot(&precomputation.SlotShare{
			PartialRoundPublicCypherKey: globals.GetGroup().GetGCyclic()})
		// Note: Share requires a batchSize of 1
		PrecompShareHandler{}.Handler(roundId, uint64(1),
			[]*services.Slot{&shareMsg})
	}

	endTime := time.Now()
	jww.INFO.Printf("Finished NewRound(RoundId: %s) in %d ms",
		clusterRoundID, (endTime.Sub(startTime))/time.Millisecond)
}

// Blocks until all given servers begin a new round
func BeginNewRound(servers []string, RoundID string) {
	startTime := time.Now()
	jww.INFO.Printf("[Last Node] Starting BeginNewRound(RoundId: %s) at %s",
		RoundID, startTime.Format(time.RFC3339))

	for i := 0; i < len(servers); {
		jww.DEBUG.Printf("Sending NewRound message to %s...\n", servers[i])
		_, err := node.SendNewRound(servers[i], &pb.InitRound{RoundID: RoundID})
		if err != nil {
			jww.ERROR.Printf("%v: Server %s failed to begin new round!\n", i,
				servers[i])
			time.Sleep(250 * time.Millisecond)
		} else {
			jww.DEBUG.Printf("%v: Server %s began new round!\n", i, servers[i])
			i++ // NOTE: we only increment on success, otherwise retry
		}
	}

	endTime := time.Now()
	jww.INFO.Printf("[Last Node] Finished BeginNewRound(RoundId: %s) in %d ms",
		RoundID, (endTime.Sub(startTime))/time.Millisecond)
}
