////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package internal

// queue.go contains the logic for the resourceQueue object
// and its interface

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/server/internal/measure"
	"gitlab.com/elixxir/server/internal/phase"
	"gitlab.com/elixxir/server/services"
	"sync/atomic"
	"testing"
	"time"
)

type ResourceQueue struct {
	activePhase phase.Phase
	phaseQueue  chan phase.Phase
	finishChan  chan phase.Phase
	timer       *time.Timer
	killChan    chan chan bool
	running     *uint32
}

//initQueue begins a queue with default channel buffer sizes
func initQueue() *ResourceQueue {
	running := uint32(0)
	return &ResourceQueue{
		// these are the phases
		phaseQueue: make(chan phase.Phase, 5000),
		// there will only active phase, and this channel is used to killChan it
		finishChan: make(chan phase.Phase, 1),
		// this channel will be used to killChan the queue
		killChan: make(chan chan bool, 1),
		running:  &running,
	}
}

// UpsertPhase adds the phase to the queue to be operated on if it is not already there
func (rq *ResourceQueue) GetPhaseQueue() chan<- phase.Phase {
	return rq.phaseQueue
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

func (rq *ResourceQueue) Kill(t time.Duration) error {
	wasRunning := atomic.SwapUint32(rq.running, 0)
	if wasRunning == 1 {
		why := make(chan bool)
		select {
		case rq.killChan <- why:
		default:
			return errors.New("Node should be able to send kill on timeout")
		}

		timer := time.NewTimer(t)
		select {
		case <-why:
			return nil
		case <-timer.C:
			return errors.New("Signal denoting kill of resource queue kill was not received")
		}
	}
	return nil
}

func (rq *ResourceQueue) run(server *Instance) {
	atomic.StoreUint32(rq.running, 1)
	rq.internalRunner(server)
	atomic.StoreUint32(rq.running, 0)
}

func (rq *ResourceQueue) internalRunner(server *Instance) {
	for true {
		//get the next phase to execute
		select {
		case why := <-rq.killChan:
			go func() { why <- true }()
			return
		case rq.activePhase = <-rq.phaseQueue:
			rq.activePhase.Measure(measure.TagActive)
		}

		jww.INFO.Printf("[%s]: RID %d Beginning execution of Phase \"%s\"", server,
			rq.activePhase.GetRoundID(), rq.activePhase.GetType())

		runningPhase := rq.activePhase

		numChunks := uint32(0)

		//Build the chunk accessor which will also increment the queue when appropriate
		var getChunk phase.GetChunk
		getChunk = func() (services.Chunk, bool) {

			nc := atomic.AddUint32(&numChunks, 1)
			if nc == 1 {
				runningPhase.Measure(measure.TagFinishFirstSlot)
			}
			chunk, ok := runningPhase.GetGraph().GetOutput()

			//Fixme: add a method to killChan this directly
			if !ok {
				//send the phase into the channel to denote it is complete
				runningPhase.UpdateFinalStates()
				rq.DenotePhaseCompletion(runningPhase)
				runningPhase.Measure(measure.TagFinishLastSlot)
			}

			return chunk, ok
		}

		curRound, err := server.GetRoundManager().GetRound(
			runningPhase.GetRoundID())

		if err != nil {
			rid := runningPhase.GetRoundID()
			roundErr := errors.Errorf("Round %d does not exist!", rid)
			server.ReportRoundFailure(roundErr, server.GetID(), rid)
			break
		}

		//start the phase's transmission handler
		handler := rq.activePhase.GetTransmissionHandler
		go func() {
			rq.activePhase.Measure(measure.TagTransmitter)
			err := handler()(runningPhase.GetRoundID(), server, getChunk, runningPhase.GetGraph().GetStream().Output)

			if err != nil {
				// This error can be used to create a Byzantine Fault
				rid := runningPhase.GetRoundID()
				roundErr := errors.Errorf("Transmission Handler for phase %s of round %v errored: %+v",
					runningPhase.GetType(), rid, err)
				server.ReportRoundFailure(roundErr, server.GetID(), rid)
			}
		}()

		//start the phase's graphs
		runningPhase.GetGraph().Run()
		//start phases's the timeout timer
		rq.timer = time.NewTimer(runningPhase.GetTimeout())

		var rtnPhase phase.Phase
		timeout := false

		select {
		case why := <-rq.killChan:
			go func() { why <- true }()
			return
		case rtnPhase = <-rq.finishChan:
		case <-rq.timer.C:
			timeout = true
		}

		//process timeout
		if timeout {
			jww.ERROR.Printf("[%v]: RID %d Graph %s of phase %s has timed out",
				server.GetID(), rq.activePhase.GetRoundID(), rq.activePhase.GetGraph().GetName(),
				rq.activePhase.GetType().String())
			rid := curRound.GetID()
			roundErr := errors.Errorf("Resource Queue has timed out killing Round %v after %s", rid, runningPhase.GetTimeout())

			server.ReportRoundFailure(roundErr, server.GetID(), rid)
			break
			//FIXME: also killChan the transmission handler
			/*kill := rq.activePhase.GetGraph().Kill()
			if kill {
				jww.ERROR.Printf("[%s]: RID %d Graph %s of phase %s killed"+
					" due to timeout",
					server, rq.activePhase.GetRoundID(), rq.activePhase.GetGraph().GetName(),
					rq.activePhase.GetType().String())
				//FIXME: send killChan round message
			} else {
				jww.FATAL.Panicf("[%s]: RID %d Graph %s of phase %s could not"+
					" be killed after timeout",
					server, rq.activePhase.GetRoundID(), rq.activePhase.GetGraph().GetName(),
					rq.activePhase.GetType().String())
			}*/
		}

		//check that the correct phase is ending
		if !rq.activePhase.Cmp(rtnPhase) {
			rid := rq.activePhase.GetRoundID()
			roundErr := errors.Errorf("INCORRECT PHASE RECEIVED phase %s of "+
				"round %v is currently running, a completion signal of %s "+
				" cannot be processed", rq.activePhase.GetType(),
				rid, rtnPhase.GetType())
			server.ReportRoundFailure(roundErr, server.GetID(), rid)
		}

		// Aggregate the runtimes of the individual threads
		adaptDur, outModsDur := runningPhase.GetGraph().GetMetrics()
		// Add this to the round dispatch duration metric
		r, _ := server.GetRoundManager().GetRound(
			rq.activePhase.GetRoundID())
		r.AddToDispatchDuration(adaptDur + outModsDur)

		jww.INFO.Printf("[%v]: RID %d Finishing execution of Phase "+
			"\"%s\" -- Adapt: %fms, outMod: %fms", server.GetID(),
			rq.activePhase.GetRoundID(), rq.activePhase.GetType(),
			float64(adaptDur.Nanoseconds()/1000000),
			float64(outModsDur.Nanoseconds()/1000000))
	}
}
