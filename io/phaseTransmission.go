package io

import (
	"gitlab.com/elixxir/comms/mixmessages"
	comm "gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/server/node"
	"gitlab.com/elixxir/server/server"
)

func TransmitPhaseForward(round *server.Round, phase node.PhaseType,
	getChunk server.GetChunk, getMessage server.GetMessage) {
	TransmitPhase(round, phase, getChunk, getMessage, true)
}

func TransmitPhaseBackward(round *server.Round, phase node.PhaseType,
	getChunk server.GetChunk, getMessage server.GetMessage) {
	TransmitPhase(round, phase, getChunk, getMessage, false)
}

func TransmitPhase(round *server.Round, phase node.PhaseType,
	getChunk server.GetChunk, getMessage server.GetMessage, direction bool) {

	batch := mixmessages.CmixBatch{}
	batch.RoundID = uint64(round.GetID())
	batch.ForPhase = int32(phase)
	batch.Slots = make([]*mixmessages.CmixSlot, round.GetBuffer().GetBatchSize())

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

	var recipient server.NodeAddress
	if direction {
		recipient = round.GetNextNodeAddress()
	} else {
		recipient = round.GetPrevNodeAddress()
	}

	comm.SendPhase(recipient.Address, recipient.Cert, &batch)
}
