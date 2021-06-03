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
	SourceId     *id.ID // Node ID of the source of the error
	sync.RWMutex
}

// Initiates a new client failure reporter.
func NewClientFailureReport(sourceID *id.ID) *ClientReport {
	m := make(map[id.Round]chan *pb.ClientError)
	return &ClientReport{
		ErrorTracker: m,
		SourceId:     sourceID,
		RWMutex:      sync.RWMutex{},
	}

	// m := make(map[id.Round]chan *pb.ClientError)
	// m[0] = make(chan *pb.ClientError, 32)
	// return &ClientReport{
	//	UserErrorTracker: m,
	// }
}

// Initializes a channel within the client error map
func (cr *ClientReport) InitErrorChan(rndID id.Round, batchSize uint32) {
	cr.RWMutex.Lock()
	newChan := make(chan *pb.ClientError, batchSize)
	cr.ErrorTracker[rndID] = newChan
	cr.RWMutex.Unlock()

}

// Sends a client error through the channel if possible
func (cr *ClientReport) Send(rndID id.Round, clientError *pb.ClientError) error {
	cr.RWMutex.RLock()
	tracker := cr.ErrorTracker[rndID]
	cr.RWMutex.RUnlock()

	// Add source ID
	clientError.Source = cr.SourceId.Marshal()

	// Send to channel
	select {
	case tracker <- clientError:
		return nil
	default:
		return errors.Errorf("Error tracker full at len %d"+
			"for round %v. Should not happen!", len(cr.ErrorTracker[rndID]), rndID)
	}

}

// Receive takes the channel (if initialized) and exhausts the channel into a list
func (cr *ClientReport) Receive(rndID id.Round) ([]*pb.ClientError, error) {
	// Read the tracker out of the map and clear it from the map
	cr.RWMutex.Lock()
	tracker := cr.ErrorTracker[rndID]
	delete(cr.ErrorTracker, rndID)
	cr.RWMutex.Unlock()

	if tracker == nil {
		return nil, errors.Errorf("Error channel for round %d non-existent", rndID)
	}

	// Exhaust the channel
	clientErrors := make([]*pb.ClientError, 0)
	for {
		select {
		case ce := <-tracker:
			clientErrors = append(clientErrors, ce)
		default:
			// Clear out channel and map entry
			return clientErrors, nil
		}
	}

}
