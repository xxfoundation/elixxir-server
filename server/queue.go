////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package server

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/server/server/phase"
	"gitlab.com/elixxir/server/services"
	"time"
)

type ResourceQueue struct {
	activePhase *phase.Phase
	phaseQueue  chan *phase.Phase
	finishChan  chan *phase.Phase
	timer       *time.Timer
}

// UpsertPhase adds the phase to the queue to be operated on if it is not already there
func (rq *ResourceQueue) UpsertPhase(p *phase.Phase) {
	if p.IncrementStateToQueued() {
		rq.phaseQueue <- p
	}
}

func queueRunner(server *Instance) {
	queue := server.GetResourceQueue()

	for true {
		//get the next phase to execute
		queue.activePhase = <-queue.phaseQueue

		//update the next phase to running
		success := queue.activePhase.IncrementStateToRunning()
		if !success {
			jww.FATAL.Panicf("Next phase %s of round %v which is queued is not in the correct state and "+
				"cannot be started", queue.activePhase.GetType().String(), queue.activePhase.GetRoundID())
		}

		runningPhase := queue.activePhase

		//Build the chunk accessor which will also increment the queue when appropriate
		var getChunk phase.GetChunk
		getChunk = func() (services.Chunk, bool) {
			chunk, ok := runningPhase.GetGraph().GetOutput()
			//Fixme: add a method to kill this directly
			if !ok {
				//send the phase into the channel to denote it is complete
				queue.finishChan <- runningPhase
			}
			return chunk, ok
		}

		//start the phase's transmission handler
		go queue.activePhase.GetTransmissionHandler()(runningPhase,
			server.GetRoundManager().GetRound(runningPhase.GetRoundID()).GetNodeAddressList(),
			getChunk, runningPhase.GetGraph().GetStream().Output)

		//start the phase's graphs
		runningPhase.GetGraph().Run()
		//start phases's the timeout timer
		queue.timer = time.NewTimer(runningPhase.GetTimeout())

		var rtnPhase *phase.Phase
		timeout := false

		//wait until a phase completes or it's timeout is reached
		select {
		case rtnPhase = <-queue.finishChan:
		case <-queue.timer.C:
			timeout = true
		}

		//process timeout
		if timeout {
			//FIXME: also kill the transmission handler
			kill := queue.activePhase.GetGraph().Kill()
			if kill {
				jww.CRITICAL.Printf("Graph %v of phase %v of round %v was killed due to timeout",
					queue.activePhase.GetGraph().GetName(), queue.activePhase.GetType().String(), queue.activePhase.GetRoundID())
				//FIXME: send kill round message
			} else {
				jww.FATAL.Panicf("Graph %v of phase %v of round %v could not be killed after timeout",
					queue.activePhase.GetGraph().GetName(), queue.activePhase.GetType().String(), queue.activePhase.GetRoundID())
			}
		}

		//check that the correct phase is ending
		if !queue.activePhase.Cmp(rtnPhase) {
			jww.FATAL.Panicf("Phase %s of round %v is currently running, "+
				"a kill message of %s cannot be processed", queue.activePhase.GetType().String(),
				queue.activePhase.GetRoundID(), rtnPhase)
		}

		//update the ending phase to the next phase which also allows the next phase in the round to run
		queue.activePhase.IncrementStateToFinished()
	}

}
