package services

import (
	"sync/atomic"
)

func NewSlotReorganizer(chIn, chOut chan *Slot,
	batchSize int) *ThreadController {

	// Create channel for receiving input if none provided
	if chIn == nil {
		chIn = make(chan *Slot, batchSize)
	}

	// Create channel for receiving output if none provided
	if chOut == nil {
		chOut = make(chan *Slot, batchSize)
	}

	// Create channel for force quitting the goroutine
	chQuit := make(chan chan bool, 1)

	// Create buffer for holding the slots
	slots := make([]*Slot, batchSize)
	reorganizedSlots := make([]*Slot, batchSize)
	sr := slotReorganizer{inChannel: chIn, outChannel: chOut,
		quitChannel: chQuit, locker: 1, batchCounter: 0,
		inSlots: slots, outSlots: reorganizedSlots}

	// Start the goroutine up
	go sr.startSlotReorganizer()

	tc := &ThreadController{InChannel: chIn, OutChannel: chOut,
		quitChannel: chQuit, threadLocker: &sr.locker}

	return tc
}

func (sr *slotReorganizer) startSlotReorganizer() {
	q := false

	var killNotify chan<- bool

	for sr.batchCounter < len(sr.inSlots) && !q {
		select {
		case in := <-sr.inChannel:
			// add a new slot in the batch
			sr.inSlots[sr.batchCounter] = in
			sr.batchCounter++
		case killNotify = <-sr.quitChannel:
			// start killing the goroutine
			q = true
		}
	}

	// put the slots in order
	if !q {
		reorganizeSlots(sr.inSlots, sr.outSlots)

		// send them out again
		for i := 0; i < len(sr.outSlots); i++ {
			sr.outChannel <- sr.outSlots[i]
		}
	}

	//close the channels
	// close(sr.inChannel)
	close(sr.outChannel)
	close(sr.quitChannel)

	// Unlock the thread locker, indicating the reorganizer is no longer running
	atomic.CompareAndSwapUint32(&sr.locker, 1, 0)

	// Notify anyone who needs to wait on the dispatcher's death
	if killNotify != nil {
		killNotify <- true
	}
}

// Used to put the message slots in their permuted order after permute phase
func reorganizeSlots(inSlots []*Slot, outSlots []*Slot) {
	for i := 0; i < len(inSlots); i++ {
		sid := (*inSlots[i]).SlotID()
		outSlots[sid] = inSlots[i]
	}
}

type slotReorganizer struct {
	noCopy noCopy

	inChannel    chan *Slot
	outChannel   chan *Slot
	quitChannel  chan chan bool
	batchCounter int
	inSlots      []*Slot
	outSlots     []*Slot
	locker       uint32
}
