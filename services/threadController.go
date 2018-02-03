package services

import (
	"sync/atomic"
)

// ThreadController is the struct which is used to externally control
// threads.
// To send data do ThreadController.InChannel <- Data
// To receive do Data <- ThreadController.OutChannel
// To force kill the dispatcher do ThreadController.QuitChannel <- true
type ThreadController struct {
	noCopy noCopy
	// Pointer to thread locker
	threadLocker *uint32

	// Channel which is used to send messages to process
	InChannel chan *Slot
	// Channel which is used to receive the results of processing
	OutChannel chan *Slot
	// Channel which is used to send and process a kill command
	quitChannel chan chan bool
}

// Determines whether the Thread is still running
func (dc *ThreadController) IsAlive() bool {
	return atomic.LoadUint32(dc.threadLocker) == 1
}

// Sends a Quit signal to the ThreadController
// Blocks until death if you pass true, doesn't block if you pass false.
func (dc *ThreadController) Kill(blockUntilDeath bool) {
	if blockUntilDeath {
		killNotify := make(chan bool)
		dc.quitChannel <- killNotify
		_ = <-killNotify
		close(killNotify)
	} else {
		dc.quitChannel <- nil
	}
}

// noCopy may be embedded into structs which must not be copied
// after the first use.
//
// See https://github.com/golang/go/issues/8005#issuecomment-190753527
// for details.
type noCopy struct{}
