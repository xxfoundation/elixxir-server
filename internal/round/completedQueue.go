///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package round

// completedQueue.go contains the round.CompletedQueue logic and its methods.
// CompletedQueue is a channel which sends a signal indicating a completed round

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/primitives/id"
)

type CompletedQueue chan *CompletedRound

const maxCompletedBatches = 100

func (cq CompletedQueue) Send(cr *CompletedRound) error {
	select {
	case cq <- cr:
		return nil
	default:
		return errors.Errorf("Completed batch queue full at len %v, "+
			"batch dropped for round %v. Check Gateway", len(cq), cr.RoundID)
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
	return make(CompletedQueue, maxCompletedBatches)
}

type CompletedRound struct {
	RoundID id.Round
	Round   []*mixmessages.Slot
}
