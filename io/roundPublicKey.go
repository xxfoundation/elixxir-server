////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package io

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/primitives/circuit"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/server/phase"
	"gitlab.com/elixxir/server/server/round"
	"gitlab.com/elixxir/server/services"
)

var ErrRoundPublicKeyTimeout = errors.New("RoundPublicKey broadcast" +
	" timed out ")

// TransmitRoundPublicKey sends the public key to every node
// in the round
func TransmitRoundPublicKey(network *node.NodeComms, batchSize uint32,
	roundID id.Round, phaseTy phase.Type, getChunk phase.GetChunk,
	getMessage phase.GetMessage, topology *circuit.Circuit,
	nodeID *id.Node) error {

	var roundPublicKeys [][]byte

	for chunk, finish := getChunk(); !finish; chunk, finish = getChunk() {
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

	for index := 1; index < topology.Len(); index++ {

		localIndex := index
		go func() {
			recipient := topology.GetNodeAtIndex(localIndex)

			ack, err := network.SendPostRoundPublicKey(recipient, roundPubKeyMsg)

			// Make sure the comm doesn't return an Ack with an
			// error message
			if ack != nil && ack.Error != "" {
				err = errors.Errorf("Remote Server Error: %s, %s",
					recipient, ack.Error)
				errChan <- err
			}
			errChan <- nil
		}()
	}

	var nonNil error

	// Wait to receive all responses from broadcast
	for index := 0; index < topology.Len()-1; index++ {
		err := <-errChan

		if err != nil {
			nonNil = err
		}
	}

	if nonNil != nil {
		return nonNil
	}

	// When all responses are received we 'send'
	// to the first node which is this node
	thisNode := topology.GetNodeAtIndex(0)

	ack, err := network.SendPostRoundPublicKey(thisNode, roundPubKeyMsg)

	// Make sure the comm doesn't return an Ack with an
	// error message
	if ack != nil && ack.Error != "" {
		err = errors.Errorf("Remote Server Error: %s, %s",
			thisNode, ack.Error)
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
