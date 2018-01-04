package services

import (
	"errors"
	"gitlab.com/privategrity/crypto/cyclic"
)

type Message struct {
	Slot uint64
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
type CryptographicOperation interface {
	run(in, out *Message, saved *[]*cyclic.Int) *Message
}

//Private struct containing the control data in the cryptopl
type dispatch struct {
	noCopy noCopy

	cryptop    CryptographicOperation
	batchSize  uint64
	saved      *[][]*cyclic.Int
	outMessage *[]*Message

	inChannel  chan *Message
	outChannel chan *Message
	quit       chan bool

	batchCntr uint64
}

//Function which actually does the dispatching
func (d *dispatch) dispatcher() {

	var out *Message

	q := false

	for (d.batchCntr < d.batchSize) && !q {

		//either process the next piece of data or quit
		select {
		case in := <-d.inChannel:

			out = (*d.outMessage)[in.Slot]

			save := &(*d.saved)[in.Slot]

			out = d.cryptop.run(in, out, save)

			d.outChannel <- out

			d.batchCntr++
		case <-d.quit:
			q = true
		}

	}

	close(d.inChannel)
	close(d.outChannel)
	close(d.quit)

}

// DispatchCryptop creates the dispatcher and returns its control structure.
// cryptop is the operation the dispatch will do
// batchSize is how many times to do the operation
// outMessage is an array for the cryptop output.
// saved is data from the server to be used in the operation
// chIn and chOut are the input and output channels, set to nil and the
// dispatcher will generate its own.
func DispatchCryptop(cryptop CryptographicOperation, batchSize uint64, outMessage *[]*Message, saved *[][]*cyclic.Int, chIn, chOut chan *Message) (*DispatchControler, error) {

	//Make sure if they want a buffered output that it is formatted correctly
	if (uint64(len(*saved)) < batchSize) || (uint64(len(*outMessage)) < batchSize) || (outMessage == nil) {
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
	d := &dispatch{cryptop: cryptop, batchSize: batchSize, saved: saved, outMessage: outMessage,
		inChannel: chIn, outChannel: chOut, quit: chQuit, batchCntr: 0}

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
