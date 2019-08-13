////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package node

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/primitives/circuit"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/io"
	"gitlab.com/elixxir/server/server"
	"gitlab.com/elixxir/server/server/phase"
	"gitlab.com/elixxir/server/server/round"
	"time"
)

// ReceiveCreateNewRound receives the create new round signal and
// creates the round
func ReceiveCreateNewRound(instance *server.Instance,
	message *mixmessages.RoundInfo) error {
	roundID := id.Round(message.ID)

	jww.INFO.Printf("[%s]: RID %d CreateNewRound RECIEVE", instance,
		roundID)

	//Build the components of the round
	phases, phaseResponses := NewRoundComponents(
		instance.GetGraphGenerator(),
		instance.GetTopology(),
		instance.GetID(),
		&instance.LastNode,
		instance.GetBatchSize())

	if len(phases) != 0 {
		phases[0].Measure("Receive Create New Round")
	}
	//Build the round
	rnd := round.New(
		instance.GetGroup(),
		instance.GetUserRegistry(),
		roundID, phases, phaseResponses,
		instance.GetTopology(),
		instance.GetID(),
		instance.GetBatchSize())
	//Add the round to the manager
	instance.GetRoundManager().AddRound(rnd)

	jww.INFO.Printf("[%s]: RID %d CreateNewRound COMPLETE", instance,
		roundID)

	return nil
}

// ReceivePostRoundPublicKey from last node and sets it for the round
// for each node. Also starts precomputation decrypt phase with a
// batch
func ReceivePostRoundPublicKey(instance *server.Instance,
	pk *mixmessages.RoundPublicKey) {

	roundID := id.Round(pk.Round.ID)
	jww.INFO.Printf("[%s]: RID %d PostRoundPublicKey START", instance,
		roundID)

	rm := instance.GetRoundManager()

	tag := phase.PrecompShare.String() + "Verification"

	r, p, err := rm.HandleIncomingComm(roundID, tag)
	if err != nil {
		jww.FATAL.Panicf("[%s]: Error on reception of "+
			"PostRoundPublicKey comm, should be able to return: \n %+v",
			instance, err)
	}
	p.Measure(tag)

	err = io.PostRoundPublicKey(instance.GetGroup(), r.GetBuffer(), pk)
	if err != nil {
		jww.FATAL.Panicf("[%s]: Error on posting PostRoundPublicKey "+
			"to io, should be able to return: %+v", instance, err)
	}

	jww.INFO.Printf("[%s]: RID %d PostRoundPublicKey PK is: %s",
		instance, roundID, r.GetBuffer().CypherPublicKey.Text(16))

	p.UpdateFinalStates()

	jww.INFO.Printf("[%s]: RID %d PostRoundPublicKey END", instance,
		roundID)

	if r.GetTopology().IsFirstNode(instance.GetID()) {
		// We need to make a fake batch here because
		// we start the precomputation decrypt phase
		// afterwards.
		// This phase needs values of 1 for the keys & cypher
		// so we can apply modular multiplication afterwards.
		// Without this the ElGamal cryptop would need to
		// support this edge case.

		batchSize := r.GetBuffer().GetBatchSize()
		blankBatch := &mixmessages.Batch{}

		blankBatch.Round = pk.Round
		blankBatch.FromPhase = int32(phase.PrecompDecrypt)
		blankBatch.Slots = make([]*mixmessages.Slot, batchSize)

		for i := uint32(0); i < batchSize; i++ {
			blankBatch.Slots[i] = &mixmessages.Slot{
				EncryptedMessageKeys:            []byte{1},
				EncryptedAssociatedDataKeys:     []byte{1},
				PartialMessageCypherText:        []byte{1},
				PartialAssociatedDataCypherText: []byte{1},
			}
		}
		decrypt, err := r.GetPhase(phase.PrecompDecrypt)
		if err != nil {
			jww.FATAL.Panicf("Error on first node PostRoundPublicKey "+
				"comm, should be able to get decrypt phase: %+v", err)
		}

		jww.INFO.Printf("[%s]: RID %d PostRoundPublicKey FIRST NODE START PHASE \"%s\"", instance,
			roundID, decrypt.GetType())

		queued :=
			decrypt.AttemptToQueue(instance.GetResourceQueue().GetPhaseQueue())

		if !queued {
			jww.FATAL.Panicf("Error on first node PostRoundPublicKey " +
				"comm, should be able to queue decrypt phase")
		}

		err = io.PostPhase(decrypt, blankBatch)

		if err != nil {
			jww.FATAL.Panicf("Error on first node PostRoundPublicKey "+
				"comm, should be able to post to decrypt phase: %+v", err)
		}
	}
}

// ReceivePostPrecompResult handles the state checks and edge checks of
// receiving the result of the precomputation
func ReceivePostPrecompResult(instance *server.Instance, roundID uint64,
	slots []*mixmessages.Slot) error {

	jww.INFO.Printf("[%s]: RID %d PostPrecompResult START", instance,
		roundID)

	rm := instance.GetRoundManager()

	tag := phase.PrecompReveal.String() + "Verification"
	r, p, err := rm.HandleIncomingComm(id.Round(roundID), tag)
	if err != nil {
		jww.FATAL.Panicf("[%s]: Error on reception of "+
			"PostPrecompResult comm, should be able to return: \n %+v",
			instance, err)
	}
	p.Measure(tag)

	err = io.PostPrecompResult(r.GetBuffer(), instance.GetGroup(), slots)
	if err != nil {
		return errors.Wrapf(err,
			"Couldn't post precomp result for round %v", roundID)
	}

	p.UpdateFinalStates()
	// Now, this round has completed this precomputation,
	// so we can push it on the precomp queue if this is the first node
	if r.GetTopology().IsFirstNode(instance.GetID()) {
		instance.GetCompletedPrecomps().Push(r)
	}
	jww.INFO.Printf("[%s]: RID %d PostPrecompResult END", instance,
		roundID)
	return nil
}

// ReceivePostPhase handles the state checks and edge checks of receiving a
// phase operation
func ReceivePostPhase(batch *mixmessages.Batch, instance *server.Instance) {
	roundID := id.Round(batch.Round.ID)
	phaseTy := phase.Type(batch.FromPhase).String()

	rm := instance.GetRoundManager()

	//Check if the operation can be done and get the correct phase if it can
	_, p, err := rm.HandleIncomingComm(roundID, phaseTy)
	if err != nil {
		jww.FATAL.Panicf("[%s]: Error on reception of "+
			"PostPhase comm, should be able to return: \n %+v",
			instance, err)
	}
	fmt.Println(p)
	tag := fmt.Sprintf("[%s]: RID %d PostPhase FROM \"%s\" FOR \"%s\" RECIEVE/START", instance,
		roundID, phaseTy, p.GetType())
	p.Measure(tag)

	jww.INFO.Printf("[%s]: RID %d PostPhase FROM \"%s\" FOR \"%s\" RECIEVE/START", instance,
		roundID, phaseTy, p.GetType())
	//queue the phase to be operated on if it is not queued yet
	p.AttemptToQueue(instance.GetResourceQueue().GetPhaseQueue())

	//HACK HACK HACK
	//The share phase needs a batchsize of 1, when it receives
	// from generation on the first node this will do the
	// conversion on the batch
	if p.GetType() == phase.PrecompShare && len(batch.Slots) != 1 {
		batch.Slots = batch.Slots[:1]
		batch.Slots[0].PartialRoundPublicCypherKey =
			instance.GetGroup().GetG().Bytes()
		jww.INFO.Printf("[%s]: RID %d PostPhase PRECOMP SHARE HACK "+
			"HACK HACK", instance, roundID)
	}

	batch.FromPhase = int32(p.GetType())

	//send the data to the phase
	err = io.PostPhase(p, batch)

	if err != nil {
		jww.FATAL.Panicf("Error on PostPhase comm, should be"+
			" able to return: %+v", err)
	}
}

// ReceiveStreamPostPhase handles the state checks and edge checks of
// receiving a phase operation
func ReceiveStreamPostPhase(streamServer mixmessages.Node_StreamPostPhaseServer,
	instance *server.Instance) error {

	batchInfo, err := node.GetPostPhaseStreamHeader(streamServer)
	if err != nil {
		return err
	}

	roundID := id.Round(batchInfo.Round.ID)
	phaseTy := phase.Type(batchInfo.FromPhase).String()

	rm := instance.GetRoundManager()

	// Check if the operation can be done and get the correct
	// phase if it can
	_, p, err := rm.HandleIncomingComm(roundID, phaseTy)
	if err != nil {
		jww.FATAL.Panicf("[%s]: Error on reception of "+
			"StreamPostPhase comm, should be able to return: \n %+v",
			instance, err)
	}
	tag := fmt.Sprintf("[%s]: RID %d StreamPostPhase FROM \"%s\" TO \"%s\" RECIEVE/START", instance,
		roundID, phaseTy, p.GetType())
	p.Measure(tag)

	jww.INFO.Printf("[%s]: RID %d StreamPostPhase FROM \"%s\" TO \"%s\" RECIEVE/START", instance,
		roundID, phaseTy, p.GetType())

	//queue the phase to be operated on if it is not queued yet
	p.AttemptToQueue(instance.GetResourceQueue().GetPhaseQueue())

	//HACK HACK HACK
	//The share phase needs a batchsize of 1, when it recieves
	// from generation on the first node this will do the
	// conversion on the batch
	if p.GetType() == phase.PrecompShare && batchInfo.BatchSize != 1 {
		batchInfo.BatchSize = 1
	}

	strmErr := io.StreamPostPhase(p, batchInfo.BatchSize, streamServer)

	return strmErr

}

// Receive PostNewBatch comm from the gateway
// This should include an entire new batch that's ready for realtime processing
func ReceivePostNewBatch(instance *server.Instance,
	newBatch *mixmessages.Batch) error {
	// This shouldn't block,
	// and should return an error if there's no round available
	// You'd want to return this error in the Ack that's available for the
	// return value of the PostNewBatch comm.
	r, ok := instance.GetCompletedPrecomps().Pop()
	if !ok {
		err := errors.New(fmt.Sprintf(
			"[%s]: ReceivePostNewBatch(): No precomputation available",
			instance))
		// This round should be at a state where its precomp
		// is complete. So, we might want more than one
		// phase, since it's at a boundary between phases.
		jww.ERROR.Print(err)
		return err
	}
	newBatch.Round = &mixmessages.RoundInfo{
		ID: uint64(r.GetID()),
	}
	newBatch.FromPhase = int32(phase.RealDecrypt)

	jww.INFO.Printf("[%s]: RID %d PostNewBatch START", instance,
		newBatch.Round.ID)

	if uint32(len(newBatch.Slots)) != r.GetBuffer().GetBatchSize() {
		jww.FATAL.Panicf("[%s]: RID %d PostNewBatch ERROR - Gateway sent "+
			"batch with improper size", instance, newBatch.Round.ID)
	}

	p, err := r.GetPhase(phase.RealDecrypt)

	if err != nil {
		jww.FATAL.Panicf(
			"[%s]: RID %d Error on incoming PostNewBatch comm, could "+
				"not find phase \"%s\": %v", instance, newBatch.Round.ID,
			phase.RealDecrypt, err)
	}

	if p.GetState() != phase.Active {
		jww.FATAL.Panicf(
			"[%s]: RID %d Error on incoming PostNewBatch comm, phase "+
				"\"%s\" at incorrect state (\"%s\" vs \"Active\")", instance,
			newBatch.Round.ID, phase.RealDecrypt, p.GetState())
	}

	// Queue the phase if it hasn't been done yet
	p.AttemptToQueue(instance.GetResourceQueue().GetPhaseQueue())

	err = io.PostPhase(p, newBatch)

	if err != nil {
		jww.FATAL.Panicf("[%s]: RID %d Error on incoming PostNewBatch comm at"+
			" io PostPhase: %+v", instance, newBatch.Round.ID, err)
	}

	// TODO send all the slot IDs that didn't make it back to the gateway
	jww.INFO.Printf("[%s]: RID %d PostNewBatch END", instance,
		newBatch.Round.ID)

	return nil
}

// Type alias for function which invokes gathering measurements
type gatherMeasureFunc func(comms *node.NodeComms, topology *circuit.Circuit, i id.Round) string

// ReceiveFinishRealtime handles the state checks and edge checks of
// receiving the signal that the realtime has completed
func ReceiveFinishRealtime(instance *server.Instance, msg *mixmessages.RoundInfo, gatherMeasure gatherMeasureFunc) error {
	//check that the round should have finished and return it
	roundID := id.Round(msg.ID)
	jww.INFO.Printf("[%s]: RID %d ReceiveFinishRealtime START",
		instance, roundID)

	rm := instance.GetRoundManager()
	//nodeComms := instance.GetNetwork()
	//topology := instance.GetTopology()

	tag := phase.RealPermute.String() + "Verification"
	r, p, err := rm.HandleIncomingComm(id.Round(roundID), tag)
	if err != nil {
		jww.FATAL.Panicf("[%s]: Error on reception of "+
			"FinishRealtime comm, should be able to return: \n %+v",
			instance, err)
	}
	p.Measure(tag)

	/*
		// Call gatherMeasure function handler if the callback is set
		// and append results to metrics log file.
		if gatherMeasure != nil {

					jww.INFO.Printf("Gather Metrics: RID %d FIRST NODE ReceiveFinishRealtime"+
				" Retrieving and storing metrics", roundID)

					measureResponse := gatherMeasure(nodeComms, topology, roundID)

					if measureResponse != "" {
				logFile := instance.GetMetricsLog()

						if logFile != "" {
					measure.AppendToMetricsLog(logFile, measureResponse)
				}
			}

				}
	*/
	p.UpdateFinalStates()

	if !instance.GetKeepBuffers() {
		jww.INFO.Printf("[%s]: RID %d ReceiveFinishRealtime CLEARING "+
			"CMIX BUFFERS", instance, roundID)

		//release the round's data
		r.GetBuffer().Erase()

		//delete the round from the manager
		rm.DeleteRound(roundID)
	} else {
		jww.WARN.Printf("[%s]: RID %d ReceiveFinishRealtime MEMORY "+
			"LEAK - Round buffers not purged ", instance,
			roundID)
	}

	jww.INFO.Printf("[%s]: RID %d ReceiveFinishRealtime END", instance,
		roundID)

	jww.INFO.Printf("[%s]: RID %d Round took %v seconds",
		instance, roundID, time.Now().Sub(r.GetTimeStart()))

	//Send the finished signal on first node
	if r.GetTopology().IsFirstNode(instance.GetID()) {
		jww.INFO.Printf("[%s]: RID %d FIRST NODE ReceiveFinishRealtime"+
			" SENDING END ROUND SIGNAL", instance, roundID)

		instance.FinishRound(roundID)

	}

	return nil
}

// ReceiveGetMeasure finds the round in msg and response with a RoundMetrics message
func ReceiveGetMeasure(instance *server.Instance, msg *mixmessages.RoundInfo) (*mixmessages.RoundMetrics, error) {
	roundID := id.Round(msg.ID)

	rm := instance.GetRoundManager()

	// Check that the round exists, grab it
	r, err := rm.GetRound(roundID)
	if err != nil {
		jww.ERROR.Printf("ERR NO ROUND FOUND WITH ID %s", msg.String())
		return nil, err
	}

	// Get information on node & topology for the metrics object
	nodeId := instance.GetID()
	topology := instance.GetTopology()
	numNodes := topology.Len()
	index := topology.GetNodeLocation(nodeId)
	resourceMonitor := instance.GetLastResourceMonitor()

	if resourceMonitor == nil {
		return nil, nil
	}

	resourceMetric := *resourceMonitor.Get()

	metrics := r.GetMeasurements(nodeId.String(), numNodes, index, resourceMetric)

	s, err := json.Marshal(metrics)

	jww.INFO.Print(s)

	ret := mixmessages.RoundMetrics{
		RoundMetricJSON: string(s),
	}

	return &ret, nil
}
