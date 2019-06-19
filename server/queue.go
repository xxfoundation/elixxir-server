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
	"testing"
	"time"
)

type ResourceQueue struct {
	activePhase phase.Phase
	phaseQueue  chan phase.Phase
	finishChan  chan phase.Phase
	timer       *time.Timer
	killChan    chan struct{}
}

//initQueue begins a queue with default channel buffer sizes
func initQueue() *ResourceQueue {
	return &ResourceQueue{
		// these are the phases
		phaseQueue: make(chan phase.Phase, 5000),
		// there will only active phase, and this channel is used to killChan it
		finishChan: make(chan phase.Phase, 1),
		// this channel will be used to killChan the queue
		killChan: make(chan struct{}),
	}
}

// UpsertPhase adds the phase to the queue to be operated on if it is not already there
func (rq *ResourceQueue) UpsertPhase(p phase.Phase) {
	if p.AttemptTransitionToQueued() {
		rq.phaseQueue <- p
	}
}

// DenotePhaseCompletion send the phase which has been completed into the queue's
// completed channel
func (rq *ResourceQueue) DenotePhaseCompletion(p phase.Phase) {
	rq.finishChan <- p
}

// GetQueue returns the internal channel used as a queue. Used in testing.
func (rq *ResourceQueue) GetQueue(t *testing.T) chan phase.Phase {
	if t == nil {
		jww.FATAL.Panicf("Queue.GetQueue is only for testing!")
	}
	return rq.phaseQueue
}

//kill the queue
func (rq *ResourceQueue) kill() {
	rq.killChan <- struct{}{}
}

func (rq *ResourceQueue) run(server *Instance) {
	for true {
		//get the next phase to execute
		select {
		case rq.activePhase = <-rq.phaseQueue:
		case <-rq.killChan:
			return
		}

		//update the next phase to running
		rq.activePhase.TransitionToRunning()

		runningPhase := rq.activePhase

		//Build the chunk accessor which will also increment the queue when appropriate
		var getChunk phase.GetChunk
		getChunk = func() (services.Chunk, bool) {
			chunk, ok := runningPhase.GetGraph().GetOutput()
			//Fixme: add a method to killChan this directly
			if !ok {
				//send the phase into the channel to denote it is complete
				rq.DenotePhaseCompletion(runningPhase)
			}
			return chunk, ok
		}

		curRound, err := server.GetRoundManager().GetRound(
			runningPhase.GetRoundID())

		if err != nil {
			jww.FATAL.Panicf("Round %d does not exist!",
				runningPhase.GetRoundID())
		}

		//start the phase's transmission handler
		handler := rq.activePhase.GetTransmissionHandler
		go func() {

			err := handler()(server.GetNetwork(), curRound.GetBuffer().GetBatchSize(),
				runningPhase.GetRoundID(),
				runningPhase.GetType(), getChunk, runningPhase.GetGraph().GetStream().Output,
				curRound.GetTopology(),
				server.GetID())

			if err != nil {
				jww.FATAL.Panicf("Transmission Handler for phase %s of round %v errored: %+v",
					runningPhase.GetType(), runningPhase.GetRoundID(), err)
			}
		}()

		//start the phase's graphs
		runningPhase.GetGraph().Run()
		//start phases's the timeout timer
		rq.timer = time.NewTimer(runningPhase.GetTimeout())

		var rtnPhase phase.Phase
		timeout := false

		//wait until a phase completes or it's timeout is reached
		completed := false

		for !completed {
			select {
			case rtnPhase = <-rq.finishChan:
			case <-rq.timer.C:
				timeout = true
			case <-rq.killChan:
				return
			}

			//process timeout
			if timeout {
				//FIXME: also killChan the transmission handler
				kill := rq.activePhase.GetGraph().Kill()
				if kill {
					jww.CRITICAL.Printf("Graph %v of phase %v of round %v was killed due to timeout",
						rq.activePhase.GetGraph().GetName(), rq.activePhase.GetType().String(), rq.activePhase.GetRoundID())
					//FIXME: send killChan round message
				} else {
					jww.FATAL.Panicf("Graph %v of phase %v of round %v could not be killed after timeout",
						rq.activePhase.GetGraph().GetName(), rq.activePhase.GetType().String(), rq.activePhase.GetRoundID())
				}
			}

			//check that the correct phase is ending
			if !rq.activePhase.Cmp(rtnPhase) {
				jww.FATAL.Panicf("phase %s of round %v is currently running, "+
					"a killChan message of %s cannot be processed", rq.activePhase.GetType().String(),
					rq.activePhase.GetRoundID(), rtnPhase)
			}

			//update the ending phase to the next phase which also allows the next phase in the round to run
			//fixme: what do we do about the timer if there is another step to completion
			completed = rtnPhase.UpdateFinalStates()
		}
	}
}
