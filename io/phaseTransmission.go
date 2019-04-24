package io

import (
	"errors"
	"fmt"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/server/node"
	"gitlab.com/elixxir/server/server"
	"gitlab.com/elixxir/server/services"
)

func ReceivePhase(batch *mixmessages.CmixBatch) error {
	roundManager := server.GetRoundManager()
	resourceQueue := server.GetResourceQueue()

	round := roundManager.GetRound(node.RoundID(batch.RoundID))

	if round == nil {
		return errors.New(fmt.Sprintf("Unknown round ID (%v) cannot continue", batch.RoundID))
	}

	if batch.ForPhase < 0 || batch.ForPhase > int32(node.NUM_PHASES) {
		return errors.New(fmt.Sprintf("Unknown phase (%v) cannot continue", batch.ForPhase))
	}

	phaseType := node.PhaseType(uint32(batch.ForPhase))

	phase := round.GetPhase(phaseType)

	if !phase.ReadyToReceiveData() {
		return errors.New(fmt.Sprintf("Phase %v of round %v is not ready to recieve", phase.Phase.String(), round.GetID()))
	}

	resourceQueue.UpsertPhase(phase)

	//Fixme: merge node.CommStream and services.Stream, this is disgusting
	commStream := interface{}(phase.Graph.GetStream()).(node.CommsStream)

	for index, messages := range batch.Slots {
		err := commStream.Input(uint32(index), messages)
		if err != nil {
			return err
		}
		//Fixme: send in larger batches
		phase.Graph.Send(services.Chunk{uint32(index), uint32(index + 1)})
	}

	return nil

}
