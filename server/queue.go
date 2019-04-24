package server

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/server/node"
	"gitlab.com/elixxir/server/services"
	"time"
)

var rq ResourceQueue

func GetResourceQueue() *ResourceQueue {
	return &rq
}

type ResourceQueue struct {
	activePhase *Phase
	phaseQueue  chan *Phase
	finishChan  chan PhaseFingerprint
	timer       time.Timer
}

func (rq *ResourceQueue) UpsertPhase(p *Phase) {
	if rq.activePhase.Round.IncrementPhaseToQueued(p.Phase) {
		rq.phaseQueue <- p
	}

}

func (rq *ResourceQueue) FinishPhase(p *Phase) {
	rq.finishChan <- p.GetFingerprint()
}

func queueRunner(queue *ResourceQueue) {

	for true {
		var fingerprint PhaseFingerprint
		timeout := false

		//get that the phase has completed or the current phase's timeout
		select {
		case fingerprint = <-queue.finishChan:
		case <-queue.timer.C:
			timeout = true
		}

		//process timeout
		if timeout {
			//FIXME: also kill the transmission handler
			kill := queue.activePhase.Graph.Kill()
			if kill {
				jww.CRITICAL.Printf("Graph %v of phase %v of round %v was killed due to timeout",
					queue.activePhase.Graph.GetName(), queue.activePhase.Phase.String(), queue.activePhase.Round.id)
				//FIXME: send kill round message
			} else {
				jww.FATAL.Panicf("Graph %v of phase %v of round %v could not be killed after timeout",
					queue.activePhase.Graph.GetName(), queue.activePhase.Phase.String(), queue.activePhase.Round.id)
			}
		}

		//check that the correct phase is ending
		if !queue.activePhase.HasFingerprint(fingerprint) {
			jww.FATAL.Panicf("Phase %s of round %v is currently running, "+
				"a kill message of phase %s of %v cannot be processed", queue.activePhase.Phase.String(),
				queue.activePhase.Round.id, fingerprint.phase.String(), fingerprint.round)
		}

		//update the ending phase to the next phase
		queue.activePhase.Round.FinishPhase(queue.activePhase.Phase)

		//get the next phase to execute
		queue.activePhase = <-queue.phaseQueue

		//update the next phase to running
		success := queue.activePhase.Round.IncrementPhaseToRunning(queue.activePhase.Phase)
		if !success {
			jww.FATAL.Panicf("Next phase %s of round %v which is queued is not in the correct state and "+
				"cannot be started", queue.activePhase.Phase.String(), queue.activePhase.Round.id)
		}

		runningPhase := queue.activePhase

		var getSlot GetSlot
		getSlot = func() (services.Chunk, bool) {
			chunk, ok := queue.activePhase.Graph.GetOutput()
			//Fixme: add a method to kill this directly
			if !ok {
				queue.FinishPhase(runningPhase)
			}
			return chunk, ok
		}

		//Fixme: merge node.CommStream and services.Stream, this is disgusting
		commStream := interface{}(runningPhase.Graph.GetStream()).(node.CommsStream)

		go queue.activePhase.TransmissionHandler(runningPhase.Round, runningPhase.Phase, getSlot, commStream.Output)
		queue.activePhase.Graph.Run()

	}

}
