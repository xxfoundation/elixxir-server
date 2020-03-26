package round

import (
	"errors"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/primitives/id"
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
	Round   []*mixmessages.Slot
}

