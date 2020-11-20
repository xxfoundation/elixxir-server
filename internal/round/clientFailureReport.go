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

const maxClientFailures = 100

// Reports client errors occurring during a round
type ClientReport struct {
	userErrorChannel chan *pb.ClientErrors
	userErrorTracker map[uint64]*pb.ClientErrors
}

// Initiates a new client failure reporter.
func NewClientFailureReport() *ClientReport {
	return &ClientReport{
		userErrorChannel: make(chan *pb.ClientErrors, maxClientFailures),
		userErrorTracker: make(map[uint64]*pb.ClientErrors),
	}
}

// Sends a client error through the channel if possible. Pulls the errors from the map
// and nils out that entry. If the map is not recognized, either no errors exist or node
// was not part of that round
func (cr *ClientReport) Send(rndID uint64) error {

	report := cr.userErrorTracker[rndID]

	if report != nil {
		select {
		case cr.userErrorChannel <- report:
			// Clean up map
			cr.userErrorTracker[rndID] = nil
			return nil
		default:
			return errors.New("Round Queue is full")
		}
	}

	return nil
}

// Report places the reported client errors in the tracker (a map) which may be pulled out
// upon completion of the given round
func (cr *ClientReport) Report(clientErrors *pb.ClientErrors, roundID uint64) {
	cr.userErrorTracker[roundID] = clientErrors
}

// Receives any client errors from the channel
//  if available.
func (cr *ClientReport) Receive() (*pb.ClientErrors, error) {
	select {
	case ce := <-cr.userErrorChannel:
		return ce, nil
	default:
		return nil, errors.New("Client reporter has nothing in it")
	}
}
