// Package services contains a dispatcher interface and functions which
// facilitate communication between the different cryptop phases.
package services

import (
	"gitlab.com/privategrity/crypto/cyclic"
	"reflect"
	"sync/atomic"
)

// Struct which contains a chunk of cryptographic data to be operated on
type Slot interface {
	//Slot of the message
	SlotID() uint64
}

//Holds keys which are used in the operation
type NodeKeys interface{}

// DispatchController is the struct which is used to externally control
//  the dispatcher
// To send data do DispatchController.InChannel <- Data
// To receive do Data <- DispatchController.OutChannel
// To force kill the dispatcher do DispatchController.QuitChannel <- true
type DispatchController struct {
	noCopy noCopy
	// Pointer to dispatch locker
	dispatchLocker *uint32

	// Channel which is used to send messages to process
	InChannel chan *Slot
	// Channel which is used to receive the results of processing
	OutChannel <-chan *Slot
	// Channel which is used to send and process a kill command
	quitChannel chan chan bool
}

// Determines whether the Dispatcher is still running
func (dc DispatchController) IsAlive() bool {
	return atomic.LoadUint32(dc.dispatchLocker) == 1
}

// Sends a Quit signal to the DispatchController
// Blocks until death if you pass true, doesn't block if you pass false.
func (dc DispatchController) Kill(blockUntilDeath bool) {
	if blockUntilDeath {
		killNotify := make(chan bool)
		dc.quitChannel <- killNotify
		_ = <-killNotify
		close(killNotify)
	} else {
		dc.quitChannel <- nil
	}
}

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

	// Locker for determining whether the dispatcher is still running
	// 1 = True, 0 = False
	locker uint32
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

			inputs[1] = reflect.ValueOf((*in))
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
	close(d.inChannel)
	close(d.outChannel)
	close(d.quit)

	// Unlock the dispatch locker, indicating the dispatcher is no longer running
	atomic.CompareAndSwapUint32(&d.locker, 1, 0)

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
func DispatchCryptop(g *cyclic.Group, cryptop CryptographicOperation, chIn, chOut chan *Slot, face interface{}) *DispatchController {

	db := cryptop.Build(g, face)

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
	d := &dispatch{cryptop: cryptop, DispatchBuilder: *db,
		inChannel: chIn, outChannel: chOut, quit: chQuit, batchCntr: 0, locker: 1}

	//runs the dispatcher
	go d.dispatcher()

	//creates the  dispatch control structure
	dc := &DispatchController{InChannel: chIn, OutChannel: chOut, quitChannel: chQuit,
		dispatchLocker: &d.locker}

	return dc

}

// noCopy may be embedded into structs which must not be copied
// after the first use.
//
// See https://github.com/golang/go/issues/8005#issuecomment-190753527
// for details.
type noCopy struct{}
