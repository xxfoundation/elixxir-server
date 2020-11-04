///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////
package round

import (
	"github.com/pkg/errors"
	pb "gitlab.com/elixxir/comms/mixmessages"
)

// Contains logic that handles an invalid client error within realtime
// Reports this error through a channel to permissioning

const MaxSimultaneousRoundErrors = 1

// Reports client errors occurring during a round
type ClientReport chan []*pb.ClientError

// Initiates a new client failure reporter.
func NewClientFailureReport() ClientReport {
	return make(chan []*pb.ClientError, MaxSimultaneousRoundErrors)
}

// Sends a client error through the channel if possible
func (cr ClientReport) Send(clientError []*pb.ClientError) error {
	select {
	case cr <- clientError:
		return nil
	default:
		return errors.New("Client reporter is full")
	}
}

// Receives any client errors from the channel
//  if available.
func (cr ClientReport) Receive() ([]*pb.ClientError, error) {
	select {
	case ce := <-cr:
		return ce, nil
	default:
		return nil, errors.New("Client reporter is has nothing in it")
	}
}
