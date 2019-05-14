package io

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/server/server/round"
)

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
