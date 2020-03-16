package round

import (
	"errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/server/phase"
	"gitlab.com/elixxir/server/services"
)

type CompletedQueue chan *CompletedRound

func (cq CompletedQueue) Send(cr *CompletedRound) error {
	select {
	case cq <- cr:
		return nil
	default:
		return errors.New("Completed batch queue full, " +
			"batch dropped. Check Gateway")
	}
}

func (cq CompletedQueue) Receive() (*CompletedRound, error) {
	select {
	case cr := <-cq:
		jww.FATAL.Printf("We have shit in the chan")
		return cr, nil
	default:
		return nil, errors.New("Did not recieve a completed round")
	}
}

func NewCompletedQueue() CompletedQueue {
	return make(CompletedQueue, 1)
}

type CompletedRound struct {
	RoundID    id.Round
	Receiver   chan services.Chunk
	GetMessage phase.GetMessage
}

func (cr *CompletedRound) GetChunk() (services.Chunk, bool) {
	chunk, ok := <-cr.Receiver
	return chunk, ok
}
