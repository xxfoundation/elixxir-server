package services

import (
	"errors"
	"gitlab.com/privategrity/crypto/cyclic"
)

type Message struct {
	Slot uint32
	Data []*cyclic.Int
}

// DispatchControler is the struct which is used to externally control the dispatcher
// To send data do DispatchControler.InChannel <- Data
// To Receive Data do data DispatchControler.OutChannel -> Data
// To force kill the dispatcher do DispatchControler.QuitChannel <- true
type DispatchControler struct {
	noCopy noCopy

	InChannel   chan<- *Message
	OutChannel  <-chan *Message
	QuitChannel chan<- bool
}

// Cryptop is the interface which contains the cryptop
// In contains the input message, out contains the output message,
// saved data are params passed by the server,
// and perm is the permutation used (if any)
type CryptographicOperation interface {
	run(in, out *Message, saved []*cyclic.Int, perm *[]uint64) *Message
}

//Private struct containing the control data in the cryptopl
type dispatch struct {
	noCopy noCopy

	cryptop   CryptographicOperation
	batchSize uint64
	saved     *[][]*cyclic.Int
	outMem    *[]*Message
	perm      *[]uint64

	inChannel  <-chan *Message
	outChannel chan<- *Message
	quit       <-chan bool

	slotCntr uint64
}

//Function which actually does the dispatching
func (d *dispatch) dispatcher() {

	var out *Message

	for d.slotCntr < d.batchSize {

		//either process the next piece of data or quit
		select {
		case in := <-d.inChannel:

			//if there is a dedicated output buffer use that, otherwise use the input buffer for output
			if d.outMem == nil {
				out = d.cryptop.run(in, in, (*d.saved)[in.Slot], d.perm)
			} else {
				out = d.cryptop.run(in, (*d.outMem)[in.Slot], (*d.saved)[in.Slot], d.perm)
			}

			d.outChannel <- out

			d.slotCntr++
		case <-d.quit:
			return
		}

	}

}

// DispatchCryptop creates the dispatcher and returns its control structure.
// cryptop is the operation the dispatch will do
// batchSize is how many times to do the operation
// outMessage is an array for the cryptop output.  set to nil to use the input.
// saved is data from the server to be used in the operation
// perm is the permutation, set to nil if unused
// chIn and chOut are the input and output channels, set to nil and the
// dispatcher will generate its own.
func DispatchCryptop(cryptop CryptographicOperation, batchSize uint64, outMessage *[]*Message, saved *[][]*cyclic.Int, perm *[]uint64, chIn, chOut chan *Message) (*DispatchControler, error) {

	//Make sure if they want a buffered output that it is formatted correctly
	if uint64(len(*saved)) < batchSize {
		// TODO: add logging as well
		return nil, errors.New("Dispatch: Improperly formatted dispatch creation")
	}

	//Creates a channel for input if none is provided
	if chIn == nil {
		chIn = make(chan *Message, batchSize)
	}

	//Creates a channel for output if none is provided
	if chOut == nil {
		chOut = make(chan *Message, batchSize)
	}

	//Creates a channel for force quitting the dispatched operation
	chQuit := make(chan bool, 1)

	//Creates the internal dispatch structure
	d := &dispatch{cryptop: cryptop, batchSize: batchSize, saved: saved, outMem: outMessage, perm: perm,
		inChannel: chIn, outChannel: chOut, quit: chQuit, slotCntr: 0}

	//runs the dispatcher
	go d.dispatcher()

	//creates the  dispatch control structure
	dc := &DispatchControler{InChannel: chIn, OutChannel: chOut, QuitChannel: chQuit}

	return dc, nil

}

// noCopy may be embedded into structs which must not be copied
// after the first use.
//
// See https://github.com/golang/go/issues/8005#issuecomment-190753527
// for details.
type noCopy struct{}
