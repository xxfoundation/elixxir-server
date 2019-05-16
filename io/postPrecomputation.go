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
	"sync"
)

// The last node (?) transmits the precomputation to all nodes but the first,
// then the first node, after precomp strip
// TODO Set this as the transmission handler for precomp strip?
func TransmitPostPrecompResult(network *node.NodeComms, batchSize uint32,
	roundID id.Round, phaseTy phase.Type, getChunk phase.GetChunk,
	getMessage phase.GetMessage, topology *circuit.Circuit,
	nodeID *id.Node) error {
	var wg sync.WaitGroup

	errChan := make(chan error, topology.Len()-1)
	// Build the message containing the precomputations
	slots := make([]*mixmessages.Slot, batchSize)
	for i := uint32(0); i < batchSize; i++ {
		slots[i] = getMessage(i)
	}

	// Send to all nodes but the first (including this one, which is the last node)
	//panic(topology.Len())
	for i := 1; i < topology.Len(); i++ {
		wg.Add(1)
		go func(index int) {
			recipient := topology.GetNodeAtIndex(index)
			ack, err := network.SendPostPrecompResult(recipient, uint64(roundID), slots)
			if err != nil {
				errChan <- errors.Wrapf(err, "")
			}
			if ack != nil && ack.Error != "" {
				errChan <- errors.Errorf("Remote error: %v", ack.Error)
			}

			wg.Done()
		}(i)
	}
	wg.Wait()

	// Return any errors at this point
	var errs error
	for len(errChan) > 0 {
		err := <-errChan
		if err != nil {
			if errs != nil {
				errs = errors.Wrap(errs, err.Error())
			} else {
				errs = err
			}
		}
	}
	if errs != nil {
		return errs
	}

	// If we got here, there weren't errors, so let's send to the first node
	// so the round can go on the finished precomps queue on that node
	recipient := topology.GetNodeAtIndex(0)
	ack, err := network.SendPostPrecompResult(recipient, uint64(roundID), slots)
	if err != nil {
		return err
	} else if ack != nil && ack.Error != "" {
		return errors.Errorf("Remote error: %v", ack.Error)
	} else {
		return nil
	}
}

func PostPrecompResult(r *round.Buffer, grp *cyclic.Group,
	slots []*mixmessages.Slot) error {
	batchSize := r.GetBatchSize()
	if batchSize != uint32(len(slots)) {
		return errors.New("PostPrecompResult: The number of slots we got" +
			" wasn't equal to the number of slots in the buffer")
	}
	overwritePrecomps(r, grp, slots)

	return nil
}

// Is this overwriting the correct fields?
func overwritePrecomps(buf *round.Buffer, grp *cyclic.Group, slots []*mixmessages.Slot) {
	for i := uint32(0); i < uint32(len(slots)); i++ {
		ADPrecomputation := buf.ADPrecomputation.Get(i)
		MessagePrecomputation := buf.MessagePrecomputation.Get(i)
		grp.SetBytes(ADPrecomputation, slots[i].PartialAssociatedDataCypherText)
		grp.SetBytes(MessagePrecomputation, slots[i].PartialMessageCypherText)
	}
}
