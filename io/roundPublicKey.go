////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package io

import (
	"fmt"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/comms/connect"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/server/measure"
	"gitlab.com/elixxir/server/server/phase"
	"gitlab.com/elixxir/server/server/round"
	"gitlab.com/elixxir/server/services"
	"sync"
)

// TransmitRoundPublicKey sends the public key to every node
// in the round
func TransmitRoundPublicKey(network *node.Comms, batchSize uint32,
	roundID id.Round, phaseTy phase.Type, getChunk phase.GetChunk,
	getMessage phase.GetMessage, topology *connect.Circuit,
	nodeID *id.Node, measureFunc phase.Measure) error {

	var roundPublicKeys [][]byte

	for chunk, finish := getChunk(); finish; chunk, finish = getChunk() {
		for i := chunk.Begin(); i < chunk.End(); i++ {
			msg := getMessage(i)
			roundPublicKeys = append(roundPublicKeys, msg.PartialRoundPublicCypherKey)
		}
	}

	if len(roundPublicKeys) != 1 {
		panic("Round public keys buffer slice contains invalid data")
	}

	// Create the message structure to send the messages
	roundPubKeyMsg := &mixmessages.RoundPublicKey{
		Round: &mixmessages.RoundInfo{
			ID: uint64(roundID),
		},
		Key: roundPublicKeys[0],
	}

	// Send public key to all nodes except the first node
	errChan := make(chan error, topology.Len()-1)

	var wg sync.WaitGroup

	measureFunc(measure.TagTransmitLastSlot)

	for index := 1; index < topology.Len(); index++ {

		localIndex := index
		wg.Add(1)
		go func() {
			// Pull the particular server host object from the commManager
			recipient := topology.GetHostAtIndex(localIndex)

			//Send the message to that node
			ack, err := network.SendPostRoundPublicKey(recipient, roundPubKeyMsg)

			if err != nil {
				errChan <- err
			}

			if ack != nil && ack.Error != "" {
				err = errors.Errorf("Remote Server Error: %s", ack.Error)
				errChan <- err
			}

			wg.Done()
		}()
	}

	// Wait to receive all responses from broadcast
	wg.Wait()

	// Return all node comms or ack errors if any
	// as a single error message
	var errs error
	for len(errChan) > 0 {
		err := <-errChan
		if errs != nil {
			errs = errors.Wrap(errs, err.Error())
		} else {
			errs = err
		}
	}
	if errs != nil {
		return errs
	}

	// When all responses are receivedFinishRealtime we 'send'
	// to the first node which is this node
	recipientID := topology.GetNodeAtIndex(0).String()
	// Pull the particular server host object from the commManager
	recipient, ok := network.GetHost(recipientID)
	if !ok {
		errMsg := fmt.Sprintf("Could not find cMix server %s in comm manager", recipientID)
		return errors.New(errMsg)
	}

	//Send message to first node
	ack, err := network.SendPostRoundPublicKey(recipient, roundPubKeyMsg)

	// Make sure the comm doesn't return an Ack with an
	// error message
	if ack != nil && ack.Error != "" {
		err = errors.Errorf("Remote Server Error: %s, %s",
			recipientID, ack.Error)
		return err
	}

	return nil
}

// PostRoundPublicKey is a comms handler which
// sets the round buffer public key if that
// key is in the group, otherwise it returns an error
func PostRoundPublicKey(grp *cyclic.Group, roundBuff *round.Buffer, pk *mixmessages.RoundPublicKey) error {

	inside := grp.BytesInside(pk.GetKey())

	if !inside {
		return services.ErrOutsideOfGroup
	}

	grp.SetBytes(roundBuff.CypherPublicKey, pk.GetKey())

	return nil
}
