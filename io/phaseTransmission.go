package io

/*
import (
	"gitlab.com/elixxir/comms/mixmessages"
	comm "gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/server/server/phase"
	"gitlab.com/elixxir/server/services"
)

func TransmitPhaseForward(phase *phase.Phase, nal *services.NodeAddressList,
	getChunk phase.GetChunk, getMessage phase.GetMessage) error {
	return TransmitPhase(phase, getChunk, getMessage, nal.GetNextNodeAddress())
}

func TransmitPhaseBackward(phase *phase.Phase, nal *services.NodeAddressList,
	getChunk phase.GetChunk, getMessage phase.GetMessage) error {
	return TransmitPhase(phase, getChunk, getMessage, nal.GetPrevNodeAddress())
}

func TransmitPhase(phase *phase.Phase, getChunk phase.GetChunk,
	getMessage phase.GetMessage, recipient services.NodeAddress) error {

	batch := mixmessages.Batch{}
	batch.Round.ID = uint64(phase.GetRoundID())
	batch.ForPhase = int32(phase.GetType())
	batch.Slots = make([]*mixmessages.Slot, phase.GetGraph().GetBatchSize())

	for true {
		chunk, finish := getChunk()
		if !finish {
			for i := chunk.Begin(); i < chunk.End(); i++ {
				batch.Slots[i] = getMessage(i)
			}
		} else {
			break
		}
	}

	_, err := comm.SendPhase(recipient.Address, recipient.Cert, &batch)
	return err
}*/
