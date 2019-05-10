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
		//panic here
	}

	// Create the message structure to send the messages
	roundPubKeyMsg := &mixmessages.RoundPublicKey{
		Round: &mixmessages.RoundInfo{
			ID: uint64(roundID),
		},
		Key: roundPublicKeys[0],
	}

	// Send public key to all nodes
	for index := 0; index < topology.Len(); index++ {

		receipient := topology.GetNodeAtIndex(index)

		ack, err := network.SendPostRoundPublicKey(receipient, roundPubKeyMsg)

		// Make sure the comm doesn't return an Ack with an
		// error message
		if ack != nil && ack.Error != "" {
			err = errors.Errorf("Remote Server Error: %s, %s",
				receipient, ack.Error)
			return err
		}
	}

	return nil
}

// PostRoundPublicKey implements the server gRPC reception handler
// for posting a public key to the round Transmission handler
func PostRoundPublicKey(grp *cyclic.Group, roundBuff *round.Buffer, pk *mixmessages.RoundPublicKey) error {

	inside := grp.BytesInside(pk.GetKey())

	if !inside {
		return services.ErrOutsideOfGroup
	}

	grp.SetBytes(roundBuff.CypherPublicKey, pk.GetKey())

	return nil
}
