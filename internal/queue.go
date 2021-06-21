///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

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
	kill        chan interface{}
	stop        chan interface{}
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
		// this channel will be used to stop the current operation
		stop:    make(chan interface{}),
		kill:    make(chan interface{}),
		running: &running,
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

// StopActivePhase the current phase from executing
func (rq *ResourceQueue) StopActivePhase(timeout time.Duration) error {
	select {
	case rq.stop <- struct{}{}:
		return nil
	case <-time.After(timeout):
		return errors.Errorf("StopActivePhase of current resource queue phase failed after %s", timeout)
	}
}

func (rq *ResourceQueue) Kill(timeout time.Duration) error {
	select {
	case rq.kill <- struct{}{}:
		return nil
	case <-time.After(timeout):
		return errors.Errorf("Resource queue kill failed after %s", timeout)
	}
}

func (rq *ResourceQueue) run(server *Instance) {
	atomic.StoreUint32(rq.running, 1)
	rq.internalRunner(server)
	atomic.StoreUint32(rq.running, 0)
}

func (rq *ResourceQueue) internalRunner(server *Instance) {
	for true {
		/*PHASE 1: wait for a phase to execute*/
		//get the next phase to execute
		rq.activePhase = nil
		for rq.activePhase == nil {
			select {
			case rq.activePhase = <-rq.phaseQueue:
				rq.activePhase.Measure(measure.TagActive)
			case <-rq.kill:
				return
			case <-rq.stop:
				continue
			}
		}

		jww.INFO.Printf("[%s]: RID %d Beginning execution of Phase \"%s\"", server,
			rq.activePhase.GetRoundID(), rq.activePhase.GetType())

		/* PHASE 2: prepare the execution*/
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
				err := runningPhase.UpdateFinalStates()
				if err != nil {
					server.ReportRoundFailure(err, server.GetID(), runningPhase.GetRoundID(), true)
				}
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
			server.ReportRoundFailure(roundErr, server.GetID(), rid, false)
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
				server.ReportRoundFailure(roundErr, server.GetID(), rid, false)
			}
		}()

		/* PHASE 3: Execute the phase */

		//start the phase's graphs
		runningPhase.GetGraph().Run()

		/* PHASE 4: Wait for results */
		//start phases' the timeout timer
		rq.timer = time.NewTimer(runningPhase.GetTimeout())

		var rtnPhase phase.Phase
		timeout := false

		select {
		case <-rq.kill:
			return
		case <-rq.stop:
			ok := rq.activePhase.GetGraph().Kill()
			if !ok {
				jww.WARN.Printf("RID %d Failed to kill graph for phase %+v", rq.activePhase.GetRoundID(), rq.activePhase.GetType().String())
			}
			continue
		case rtnPhase = <-rq.finishChan:
		case <-rq.timer.C:
			timeout = true
		}

		/* PHASE 5: Handle results */
		// process timeout
		if timeout {
			jww.ERROR.Printf("[%v]: RID %d Graph %s of phase %s has timed out",
				server.GetID(), rq.activePhase.GetRoundID(), rq.activePhase.GetGraph().GetName(),
				rq.activePhase.GetType().String())
			rid := curRound.GetID()
			roundErr := errors.Errorf("Resource Queue has timed out killing Round %v after %s", rid, runningPhase.GetTimeout())

			kill := rq.activePhase.GetGraph().Kill()
			if !kill {
				jww.ERROR.Printf("[%s]: RID %d Graph %s of phase %s killed"+
					" due to timeout",
					server, rq.activePhase.GetRoundID(), rq.activePhase.GetGraph().GetName(),
					rq.activePhase.GetType().String())
				server.ReportRoundFailure(roundErr, server.GetID(), rid, true)
			}
			server.ReportRoundFailure(roundErr, server.GetID(), rid, false)
			continue
		}

		//check that the correct phase is ending
		if !rq.activePhase.Cmp(rtnPhase) {
			rid := rq.activePhase.GetRoundID()
			roundErr := errors.Errorf("INCORRECT PHASE RECEIVED phase %s of "+
				"round %v is currently running, a completion signal of %s "+
				" cannot be processed", rq.activePhase.GetType(),
				rid, rtnPhase.GetType())
			server.ReportRoundFailure(roundErr, server.GetID(), rid, false)
		}

		// Aggregate the runtimes of the individual threads
		adaptDur, outModsDur := runningPhase.GetGraph().GetMetrics()
		// Add this to the round dispatch duration metric
		r, _ := server.GetRoundManager().GetRound(
			rq.activePhase.GetRoundID())
		r.AddToDispatchDuration(adaptDur + outModsDur)

		jww.INFO.Printf("[%v]: RID %d Finishing execution of Phase "+
			"\"%s\" -- Adapt: %s, outMod: %s", server.GetID(),
			rq.activePhase.GetRoundID(), rq.activePhase.GetType(),
			adaptDur, outModsDur)
	}
}
