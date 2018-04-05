////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package services

import (
	"sync/atomic"
)

// ThreadController is the struct which is used to externally control
// threads.
// To send data do ThreadController.InChannel <- Data
// To receive do Data <- ThreadController.OutChannel
// To force kill the thread controller do ThreadController.Kill(false)
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
	//Number of threads its controlling
	numThreads uint32
}

// Determines whether the Thread is still running
func (dc *ThreadController) IsAlive() bool {
	return atomic.LoadUint32(dc.threadLocker) > 0
}

// Sends a Quit signal to the ThreadController
// Blocks until death if you pass true, doesn't block if you pass false.
func (dc *ThreadController) Kill(blockUntilDeath bool) {
	// this makes it so killing works if they dont set numthreads,
	// as in some older implementations
	if dc.numThreads == 0 {
		dc.numThreads = 1
	}

	// I am proud of how horrible this hack is
	for i := uint32(0); i < dc.numThreads; i++ {
		if blockUntilDeath {
			killNotify := make(chan bool)
			dc.quitChannel <- killNotify
			_ = <-killNotify
			close(killNotify)
		} else {
			dc.quitChannel <- nil
		}
	}

}

// noCopy may be embedded into structs which must not be copied
// after the first use.
//
// See https://github.com/golang/go/issues/8005#issuecomment-190753527
// for details.
type noCopy struct{}
