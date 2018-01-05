package services

import (
	"gitlab.com/privategrity/crypto/cyclic"
)

// Struct which contains a chunck of cryptographic data to be operated on
type Message struct {
	//Slot of the message
	Slot uint64
	//Data contained within the message
	Data []*cyclic.Int
}

// Creates a new mesage with a datasize of the given width filled with
// globals.Max4192BitInt
func NewMessage(slot, width uint64, val *cyclic.Int) *Message {
	ml := make([]*cyclic.Int, width)

	i := uint64(0)
	for i < width {
		ml[i] = cyclic.NewInt(0)
		ml[i].SetBytes(cyclic.Max4kBitInt)
		if val != nil {
			ml[i].Set(val)
		}
		i++

	}

	return &Message{Slot: slot, Data: ml}
}

// Width returns the width of the given message
func Width(m *Message) uint64 {
	return uint64(len((*m).Data))
}

// DispatchControler is the struct which is used to externally control
//  the dispatcher
// To send data do DispatchControler.InChannel <- Data
// To receive do Data <- DispatchControler.OutChannel
// To force kill the dispatcher do DispatchControler.QuitChannel <- true
type DispatchControler struct {
	noCopy noCopy

	// Channel which is used to send messages to process
	InChannel chan<- *Message
	// Channel which is used to recieve the results of processing
	OutChannel <-chan *Message
	// Channel which is used to send a kill command
	QuitChannel chan<- bool
}

// Cryptop is the interface which contains the cryptop
type CryptographicOperation interface {
	// Run is the function which exicutes the cryptogrphic operation
	// in is the data coming in to be operated on
	// out is the result of the operation, it is also returned
	// saved is the data saved on the node which is used in the operation
	run(in, out *Message, saved *[]*cyclic.Int) *Message

	// Build is used to generate the data which is used in run.
	// takes an empty interface
	build(face interface{}) *DispatchBuilder
}

// Contains the data required to configure the dispatcher and to exicute "run"
type DispatchBuilder struct {
	// Size of the batch the cryptop is to be run on
	BatchSize uint64
	// Pointers to Data from the server which is to be passed to run
	Saved *[][]*cyclic.Int
	// buffer of messages which will be used to store the results
	OutMessage *[]*Message
}

// Private struct containing the control data in the cryptop
type dispatch struct {
	noCopy noCopy

	// Interface containing Crtptographic Operation and its builder
	cryptop CryptographicOperation
	// Embeded struct containing the data used to run the cryptop
	DispatchBuilder

	// Channel used to recieve data to be processed
	inChannel chan *Message
	// Channel used to send data to be processed
	outChannel chan *Message
	// Channel used to recieve kill commands
	quit chan bool

	//Counter of how many messages have been processed
	batchCntr uint64
}

//Function which actually does the dispatching
func (d *dispatch) dispatcher() {

	var out *Message

	q := false

	runcounter := 0

	for (d.batchCntr < d.DispatchBuilder.BatchSize) && !q {

		//either process the next piece of data or quit
		select {
		case in := <-d.inChannel:
			//recieved message

			out = (*d.DispatchBuilder.OutMessage)[in.Slot]

			save := &(*d.DispatchBuilder.Saved)[in.Slot]

			//process message using the cryptop
			out = d.cryptop.run(in, out, save)

			//send the result
			d.outChannel <- out

			d.batchCntr++
		case <-d.quit:
			//kill the dispatcher
			q = true
		}

		runcounter++

	}

	//close the channels
	close(d.inChannel)
	close(d.outChannel)
	close(d.quit)

}

// DispatchCryptop creates the dispatcher and returns its control structure.
// cryptop is the operation the dispatch will do
// round is a pointer to the round object the dispatcher is in
// chIn and chOut are the input and output channels, set to nil and the
//  dispatcher will generate its own.
func DispatchCryptop(cryptop CryptographicOperation, chIn, chOut chan *Message, face interface{}) *DispatchControler {

	db := cryptop.build(face)

	//Creates a channel for input if none is provided
	if chIn == nil {
		chIn = make(chan *Message, db.BatchSize)
	}

	//Creates a channel for output if none is provided
	if chOut == nil {
		chOut = make(chan *Message, db.BatchSize)
	}

	//Creates a channel for force quitting the dispatched operation
	chQuit := make(chan bool, 1)

	//build the data used to run the cryptop

	//Creates the internal dispatch structure
	d := &dispatch{cryptop: cryptop, DispatchBuilder: *db,
		inChannel: chIn, outChannel: chOut, quit: chQuit, batchCntr: 0}

	//runs the dispatcher
	go d.dispatcher()

	//creates the  dispatch control structure
	dc := &DispatchControler{InChannel: chIn, OutChannel: chOut, QuitChannel: chQuit}

	return dc

}

// noCopy may be embedded into structs which must not be copied
// after the first use.
//
// See https://github.com/golang/go/issues/8005#issuecomment-190753527
// for details.
type noCopy struct{}
