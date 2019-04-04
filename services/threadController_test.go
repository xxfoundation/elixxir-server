////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package services

import (
	"sync/atomic"
	"testing"
)

type slottst struct {
	id uint64
}

func (s slottst) SlotID() uint64 {
	return s.id
}

func threadtest(td *ThreadController) {

	var killTc chan bool

	q := false

	for !q {

		select {

		case in := <-td.InChannel:
			td.OutChannel <- in

		case killTc = <-td.quitChannel:
			q = true
		}
	}

	close(td.InChannel)
	close(td.OutChannel)
	close(td.quitChannel)

	atomic.CompareAndSwapUint32(td.threadLocker, 1, 0)

	// Notify anyone who needs to wait on the dispatcher's death
	if killTc != nil {
		killTc <- true
	}
}

func TestThreadController(t *testing.T) {

	inputSlc := make([]slottst, 10)

	for i := 0; i < len(inputSlc); i++ {
		inputSlc[i] = slottst{id: uint64(i)}
	}

	//create thread controller
	tl := uint32(1)
	tc := &ThreadController{threadLocker: &tl, InChannel: make(chan *Slot, 10),
		OutChannel: make(chan *Slot, 10), quitChannel: make(chan chan bool, 1)}

	//run thread test
	go threadtest(tc)

	for i := 0; i < len(inputSlc); i++ {

		slt := (Slot)(inputSlc[i])
		tc.InChannel <- &slt
		out := <-tc.OutChannel

		//compare out with input slice
		if inputSlc[i].SlotID() != (*out).SlotID() {
			t.Errorf("ThreadController test failed! Expected val: %v Actual: %v", inputSlc[i], out)
		}
	}

	if !tc.IsAlive() {
		t.Errorf("IsAlive: threadController should be alive since it has just been initialized")
	}

	tc.Kill(true)

	if tc.IsAlive() {
		t.Errorf("Kill: threadController should NOT be alive after executing the kill statement")
	}

}
