// Package services contains a dispatcher interface and functions which
// facilitate communication between the different cryptop phases.
package services

import (
	"sync/atomic"
)

type BatchTransmission interface {
	Handler(roundId string, batchSize uint64, slots []*Slot)
}

type transmit struct {
	noCopy noCopy

	// Channel used to receive data to be processed
	inChannel chan *Slot

	// Channel used to receive kill commands
	quit chan chan bool

	batchSize uint64

	roundId string

	// Locker for determining whether the dispatcher is still running
	// 1 = True, 0 = False
	locker uint32
}

func (t *transmit) transmitter(bt BatchTransmission) {
	q := false

	batchCntr := uint64(0)

	var killNotify chan<- bool

	slots := make([]*Slot, t.batchSize)

	for batchCntr < t.batchSize && !q {

		//either process the next piece of data or quit

		select {
		case in := <-t.inChannel:
			slots[batchCntr] = in

			batchCntr++

		case killNotify = <-t.quit:
			//kill the dispatcher
			q = true
		}

	}

	if !q {
		bt.Handler(t.roundId, t.batchSize, slots)
	}

	//close the channels
	close(t.inChannel)
	close(t.quit)

	// Unlock the dispatch locker, indicating the dispatcher is no longer running
	atomic.CompareAndSwapUint32(&t.locker, 1, 0)

	// Notify anyone who needs to wait on the dispatcher's death
	if killNotify != nil {
		killNotify <- true
	}
}

func BatchTransmissionDispatch(roundId string, batchSize uint64, inCh chan *Slot, bt BatchTransmission) *ThreadController {

	//Creates a channel for force quitting the dispatched operation
	chQuit := make(chan chan bool, 1)

	//Creates the internal dispatch structure
	t := &transmit{inChannel: inCh, quit: chQuit, batchSize: batchSize, roundId: roundId, locker: 1}

	//runs the dispatcher
	go t.transmitter(bt)

	//creates the  dispatch control structure
	dc := &ThreadController{InChannel: inCh, OutChannel: nil, quitChannel: chQuit,
		threadLocker: &t.locker}

	return dc

}
