////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Package services contains a dispatcher interface and functions which
// facilitate communication between the different cryptop phases.
package services

import (
	"gitlab.com/elixxir/crypto/cyclic"
	"math"
	"reflect"
	"runtime"
	"sync/atomic"
)

//Holds keys which are used in the operation
type NodeKeys interface{}

// Cryptop is the interface which contains the cryptop
type CryptographicOperation interface {
	// Run is the function which executes the cryptographic operation
	// in is the data coming in to be operated on
	// out is the result of the operation, it is also returned
	// saved is the data saved on the node which is used in the operation
	// Run(g *cyclic.Group, in, out Message, saved NodeKeys) Message

	// Build is used to generate the data which is used in run.
	// takes an empty interface
	Build(g *cyclic.Group, face interface{}) *DispatchBuilder
}

// DispatchBuilder contains the data required to configure the dispatcher
// and to execute "run".
type DispatchBuilder struct {
	// Size of the batch the cryptop is to be run on
	BatchSize uint64
	// Pointers to Data from the server which is to be passed to run
	Keys *[]NodeKeys
	// buffer of messages which will be used to store the results
	Output *[]Slot
	//Group to use to execute operations
	G *cyclic.Group
}

// dispatch is a private struct containing the control data in the cryptop
type dispatch struct {
	noCopy noCopy

	// Interface containing Cryptographic Operation and its builder
	cryptop CryptographicOperation
	// Embedded struct containing the data used to run the cryptop
	DispatchBuilder

	// Channel used to receive data to be processed
	inChannel chan *Slot
	// Channel used to send data to be processed
	outChannel chan *Slot
	// Channel used to receive kill commands
	quit chan chan bool

	//Counter of how many messages have been processed
	batchCntr uint64

	// Locker for determining how many threads are still running
	locker *uint32
}

// dispatcher is the function which actually does the dispatching
func (d *dispatch) dispatcher() {
	q := false

	runFunc := reflect.ValueOf(d.cryptop).MethodByName("Run")

	inputs := make([]reflect.Value, 4)

	inputs[0] = reflect.ValueOf(d.DispatchBuilder.G)

	var killNotify chan<- bool

	for d.batchCntr < d.DispatchBuilder.BatchSize && !q {

		//either process the next piece of data or quit
		select {
		case in := <-d.inChannel:
			//received message

			out := (*d.DispatchBuilder.Output)[(*in).SlotID()]

			inputs[1] = reflect.ValueOf(*in)
			inputs[2] = reflect.ValueOf(out)
			inputs[3] = reflect.ValueOf((*d.DispatchBuilder.Keys)[(*in).SlotID()])

			//process message using the cryptop
			returnedValues := runFunc.Call(inputs)
			a := returnedValues[0].Interface()
			b := a.(Slot)

			//send the result
			d.outChannel <- &b

			d.batchCntr++
		case killNotify = <-d.quit:
			//kill the dispatcher
			q = true
		}

	}

	//close the channels
	// FIXME: This prevents double-close when chaining, perhaps senders should
	//        Always be responsible for closing their channels?
	//close(d.inChannel)

	// Unlock the dispatch locker, indicating the dispatcher is no longer running
	result := atomic.AddUint32(d.locker, ^uint32(0))

	if result == uint32(0) {
		close(d.outChannel)
		close(d.quit)
	}

	// Notify anyone who needs to wait on the dispatcher's death
	if killNotify != nil {
		killNotify <- true
	}
}

// DispatchCryptop creates the dispatcher and returns its control structure.
// cryptop is the operation the dispatch will do
// round is a pointer to the round object the dispatcher is in
// chIn and chOut are the input and output channels, set to nil and the
//  dispatcher will generate its own.
func DispatchCryptop(g *cyclic.Group, cryptop CryptographicOperation, chIn, chOut chan *Slot, face interface{}) *ThreadController {

	return DispatchCryptopSized(g, cryptop, chIn, chOut, 0, face)

}

// DispatchCryptopSized creates the dispatcher with an alternate batch size
// and returns its control structure. cryptop is the operation the dispatch
// will do round is a pointer to the round object the dispatcher is in
// chIn and chOut are the input and output channels, set to nil and the
// dispatcher will generate its own.
func DispatchCryptopSized(g *cyclic.Group, cryptop CryptographicOperation, chIn, chOut chan *Slot, batchSize uint64, face interface{}) *ThreadController {

	db := cryptop.Build(g, face)

	if batchSize != 0 {
		db.BatchSize = batchSize
	}

	//Creates a channel for input if none is provided
	if chIn == nil {
		chIn = make(chan *Slot, db.BatchSize)
	}

	//Creates a channel for output if none is provided
	if chOut == nil {
		chOut = make(chan *Slot, db.BatchSize)
	}

	//Creates a channel for force quitting the dispatched operation
	chQuit := make(chan chan bool, 1)

	//build the data used to run the cryptop

	//Creates the internal dispatch structure
	numThreads := uint32(runtime.NumCPU())

	if uint64(numThreads) > db.BatchSize {
		numThreads = uint32(db.BatchSize)
	}

	locker := uint32(numThreads)

	batchPerThread := batchsizePerThread(numThreads, db.BatchSize)

	for i := uint32(0); i < numThreads; i++ {
		//build dispatcher for thread
		d := &dispatch{cryptop: cryptop, DispatchBuilder: *db,
			inChannel: chIn, outChannel: chOut, quit: chQuit, batchCntr: 0,
			locker: &locker}
		d.BatchSize = batchPerThread[i]

		//runs the dispatcher for current thread
		go d.dispatcher()
	}

	//creates the  dispatch control structure
	dc := &ThreadController{InChannel: chIn, OutChannel: chOut, quitChannel: chQuit,
		threadLocker: &locker, numThreads: numThreads}

	return dc

}

func batchsizePerThread(numThreads uint32, batchsize uint64) []uint64 {
	base := uint64(math.Floor(float64(batchsize) / float64(numThreads)))

	batchlist := make([]uint64, numThreads)

	for i := uint32(0); i < numThreads; i++ {
		batchlist[i] = base
	}

	delta := batchsize - uint64(numThreads)*base

	for i := uint64(0); i < delta; i++ {
		batchlist[i]++
	}

	return batchlist
}
