////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package io

import (
	"gitlab.com/privategrity/comms/node"
	"gitlab.com/privategrity/server/cryptops"
	"gitlab.com/privategrity/server/cryptops/precomputation"
	"gitlab.com/privategrity/server/cryptops/realtime"
	"gitlab.com/privategrity/server/globals"
	"gitlab.com/privategrity/server/services"

	jww "github.com/spf13/jwalterweatherman"
	pb "gitlab.com/privategrity/comms/mixmessages"
	"time"
)


// Comms method for kicking off a new round in CMIX
func (s ServerImpl) NewRound(clusterRoundID string) {
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
	round := globals.NewRound(batchSize)

	// Add round to the GlobalRoundMap
	globals.GlobalRoundMap.AddRound(roundId, round)

	// Timeout this round on this node for precomputation after 10 minutes to
	// prevent deadlock
	timeoutPrecomputation(roundId, 10*time.Second)

	globals.GlobalRoundMap.GetRound(roundId).SetPhase(globals.PRECOMP_GENERATION)

	// Create the controller for PrecompShare
	// Note: Share requires a batchSize of 1
	precompShareController := services.DispatchCryptopSized(globals.Grp,
		precomputation.Share{}, nil, nil, uint64(1), round)
	// Add the inChannel from the controller to round
	round.AddChannel(globals.PRECOMP_SHARE, precompShareController.InChannel)
	// Kick off PrecompShare Transmission Handler
	// Note: Share requires a batchSize of 1
	services.BatchTransmissionDispatch(roundId, uint64(1),
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

	// Create the reception keygen for RealtimeDecrypt
	receptionKeygenInit := make([]interface{}, 2)
	receptionKeygenInit[0] = round
	receptionKeygenInit[1] = cryptops.TRANSMISSION
	realtimeReceptionKeygen := services.DispatchCryptop(globals.Grp,
		cryptops.GenerateClientKey{}, nil, nil, receptionKeygenInit)
	// Create the controller for RealtimeDecrypt
	realtimeDecryptController := services.DispatchCryptop(globals.Grp,
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
	realtimeTransmissionKeygen := services.DispatchCryptop(globals.Grp,
		cryptops.GenerateClientKey{}, nil, nil, transmissionKeygenInit)
	// Create the controller for RealtimeEncrypt
	realtimeEncryptController := services.DispatchCryptop(globals.Grp,
		realtime.Encrypt{}, realtimeTransmissionKeygen.OutChannel, nil, round)
	// Add the InChannel from the keygen controller to round
	round.AddChannel(globals.REAL_ENCRYPT, realtimeTransmissionKeygen.InChannel)
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

	// Create the dispatch controller for PrecompGeneration
	precompGenerationController := services.DispatchCryptop(globals.Grp,
		precomputation.Generation{}, nil, nil, round)
	// Run PrecompGeneration for the entire batch
	for j := uint64(0); j < batchSize; j++ {
		genMsg := services.Slot(&precomputation.SlotGeneration{Slot: j})
		precompGenerationController.InChannel <- &genMsg
		_ = <-precompGenerationController.OutChannel
	}
	globals.GlobalRoundMap.GetRound(roundId).SetPhase(globals.PRECOMP_SHARE)

	// Generate debugging information
	jww.DEBUG.Printf("R Value: %v, S Value: %v, T Value: %v",
		round.R_INV[0].Text(10),
		round.S_INV[0].Text(10),
		round.T_INV[0].Text(10))
	jww.DEBUG.Printf("U Value: %v, V Value: %v",
		round.U_INV[0].Text(10),
		round.V_INV[0].Text(10))

	if globals.IsLastNode {
		// Create the controller for RealtimeIdentify
		realtimeIdentifyController := services.DispatchCryptop(globals.Grp,
			realtime.Identify{}, nil, nil, round)
		// Add the InChannel from the controller to round
		round.AddChannel(globals.REAL_IDENTIFY,
			realtimeIdentifyController.InChannel)
		//Add Verify on the end of Identify
		realtimeVerifyController := services.DispatchCryptop(globals.Grp,
			realtime.Verify{}, realtimeIdentifyController.OutChannel, nil, round)
		// Kick off RealtimeIdentify Transmission Handler
		services.BatchTransmissionDispatch(roundId, batchSize,
			realtimeVerifyController.OutChannel, RealtimeIdentifyHandler{})

		// Create the controller for RealtimePeel
		realtimePeelController := services.DispatchCryptop(globals.Grp,
			realtime.Peel{}, nil, nil, round)
		// Add the InChannel from the controller to round
		round.AddChannel(globals.REAL_PEEL, realtimePeelController.InChannel)
		// Kick off RealtimePeel Transmission Handler
		services.BatchTransmissionDispatch(roundId, batchSize,
			realtimePeelController.OutChannel, RealtimePeelHandler{})

		// Create the controller for PrecompStrip
		precompStripController := services.DispatchCryptop(globals.Grp,
			precomputation.Strip{}, nil, nil, round)
		// Add the InChannel from the controller to round
		round.AddChannel(globals.PRECOMP_STRIP, precompStripController.InChannel)
		// Kick off PrecompStrip Transmission Handler
		services.BatchTransmissionDispatch(roundId, batchSize,
			precompStripController.OutChannel, PrecompStripHandler{})

		jww.INFO.Println("Beginning PrecompShare Phase...")
		shareMsg := services.Slot(&precomputation.SlotShare{
			PartialRoundPublicCypherKey: globals.Grp.G})
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
