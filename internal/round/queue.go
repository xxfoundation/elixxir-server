///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package round

// queue.go contains the queue type and its method. A round.Queue is a
// channeled buffer that sends round info across threads

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/comms/mixmessages"
)

type Queue chan *mixmessages.RoundInfo

func NewQueue() Queue {
	return make(chan *mixmessages.RoundInfo, 1)
}

func (rq Queue) Send(ri *mixmessages.RoundInfo) error {
	select {
	case rq <- ri:
		return nil
	default:
		return errors.New("Round Queue is full")
	}
}

func (rq Queue) Receive() (*mixmessages.RoundInfo, error) {
	select {
	case ri := <-rq:
		return ri, nil
	default:
		return nil, errors.New("Round Queue is empty")
	}
}
