package round

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/primitives/id"
	jww "github.com/spf13/jwalterweatherman"
)

type CompletedQueue chan *CompletedRound

func (cq CompletedQueue) Send(cr *CompletedRound) error {
	select {
	case cq <- cr:
		jww.INFO.Printf("Send Completed Round %v, queue len: %v", cr.RoundID, len(cq))
		return nil
	default:
		return errors.Errorf("Completed batch queue full at len %v, " +
			"batch dropped for round %v. Check Gateway", len(cq), cr.RoundID)
	}
}

func (cq CompletedQueue) Receive() (*CompletedRound, error) {
	select {
	case cr := <-cq:
		jww.INFO.Printf("Receved Completed Round %v, queue len: %v", cr.RoundID, len(cq))
		return cr, nil
	default:
		return nil, errors.New("Did not recieve a completed round")
	}
}

func NewCompletedQueue() CompletedQueue {
	return make(CompletedQueue, 100)
}

type CompletedRound struct {
	RoundID id.Round
	Round   []*mixmessages.Slot
}
