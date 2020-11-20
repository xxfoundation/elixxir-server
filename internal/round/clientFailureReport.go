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
	"gitlab.com/xx_network/primitives/id"
	"sync"
)

// Contains logic that handles an invalid client error within realtime
// Reports this error through a channel to permissioning

// Client report maps a channel containing a series of client errors
// to a round ID
type ClientReport struct {
	ErrorTracker map[id.Round]chan *pb.ClientError
	sync.RWMutex
}

// Initiates a new client failure reporter.
func NewClientFailureReport() *ClientReport {
	m := make(map[id.Round]chan *pb.ClientError)
	return &ClientReport{
		ErrorTracker: m,
		RWMutex:      sync.RWMutex{},
	}
	//m := make(map[id.Round]chan *pb.ClientError)
	//m[0] = make(chan *pb.ClientError, 32)
	//return &ClientReport{
	//	UserErrorTracker: m,
	//}
}

// Sends a client error through the channel if possible
func (cr *ClientReport) Send(rndID id.Round, err *pb.ClientError, batchSize uint32) error {
	cr.RWMutex.Lock()
	defer cr.RWMutex.Unlock()
	// Check that map entry has been initialized
	if cr.ErrorTracker[rndID] == nil {
		cr.ErrorTracker[rndID] = make(chan *pb.ClientError, batchSize)
	}
	// Send to channel
	select {
	case cr.ErrorTracker[rndID] <- err:
		return nil
	default:
		// todo: rework error message
		return errors.Errorf("Error tracker full at len %d"+
			"for round %v. Should not happen!", len(cr.ErrorTracker[rndID]), rndID)
	}

}

// Receive takes the channel (if initialized) and exhausts the channel into a list
func (cr *ClientReport) Receive(rndID id.Round) ([]*pb.ClientError, error) {
	cr.RWMutex.Lock()
	defer cr.RWMutex.Unlock()

	if cr.ErrorTracker[rndID] == nil {
		return nil, errors.Errorf("Error channel for round %d non-existent", rndID)
	}

	// Exhaust the channel
	clientErrors := make([]*pb.ClientError, 0)
	for {
		select {
		case ce := <-cr.ErrorTracker[rndID]:
			clientErrors = append(clientErrors, ce)
		default:
			// Clear out channel
			cr.RWMutex.Lock()
			cr.ErrorTracker[rndID] = nil
			cr.RWMutex.Unlock()

			return clientErrors, nil
		}
	}

}
