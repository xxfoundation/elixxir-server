////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package services

import (
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"time"
)

const (
	THREAD_INACTIVE uint32 = iota
	THREAD_WAITING
	THREAD_ACTIVE
	THREAD_COMPLETE
)

const MAX_THREADS uint8 = 64

type moduleState struct {
	numTh  uint8
	states *uint64
	// These are used to kill goroutines
	threads []chan chan bool
	locks   []sync.Mutex
}

// This doesn't seem like the best way to initialize the struct
func (ms *moduleState) Init() {
	if ms.numTh > MAX_THREADS {
		panic(fmt.Sprintf("Cannot start module with more than %v threads, started with %v", MAX_THREADS, ms.numTh))
	}
	ms.threads = make([]chan chan bool, ms.numTh)
	ms.locks = make([]sync.Mutex, ms.numTh)

	for i := uint8(0); i < ms.numTh; i++ {
		ms.threads[i] = make(chan chan bool)
	}

	states := new(uint64)

	if ms.numTh == 64 {
		*states = math.MaxUint64
	} else {
		*states = (uint64(1) << (ms.numTh + 1)) - uint64(1)
	}

	ms.states = states

}

func (ms *moduleState) denoteClose(thread uint8, killnotify chan bool) bool {
	if thread >= ms.numTh {
		return false
	}

	flag := uint64(1) << thread

	ms.locks[thread].Lock()
	state := atomic.LoadUint64(ms.states)

	if state&flag == 0 {
		ms.locks[thread].Unlock()
		return false
	}

	flagInv := ^flag

	for !atomic.CompareAndSwapUint64(ms.states, state, state&flagInv) {
		state = atomic.LoadUint64(ms.states)
	}
	if killnotify != nil {
		killnotify <- true
	}

	// Does commenting this prevent the send on closed panic?
	// It does not
	close(ms.threads[thread])

	ms.locks[thread].Unlock()

	return true
}

func (ms *moduleState) killThread(thread uint8, timeout time.Duration) bool {
	if thread >= ms.numTh {
		return false
	}

	if !ms.IsRunning(thread) {
		return true
	}

	killNotify := make(chan bool)

	killTimer := time.NewTimer(timeout)

	select {
	case <-killTimer.C:
		return false
	case <-killNotify:
	}

	flag := uint64(1) << thread
	state := atomic.LoadUint64(ms.states)

	if state&flag != 0 {
		return false
	}
	return true
}

func (ms *moduleState) IsRunning(thread uint8) bool {
	flag := uint64(1) << thread
	state := atomic.LoadUint64(ms.states)
	return state&flag != 0
}

// Why does it only check 32 bits?
func (ms *moduleState) AnyRunning() bool {
	state := atomic.LoadUint64(ms.states)
	return state&math.MaxUint32 != 0
}

func (ms *moduleState) Kill(timeout time.Duration) bool {
	success := true

	for itr, _ := range ms.threads {
		if ms.IsRunning(uint8(itr)) {
			success = success && ms.killThread(uint8(itr), timeout)
		}
	}
	return success
}
