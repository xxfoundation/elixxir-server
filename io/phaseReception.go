package io

/*
import (
	"errors"
	"fmt"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/server"
	"gitlab.com/elixxir/server/server/phase"
	"gitlab.com/elixxir/server/services"
)

func ReceivePhase(instance *server.Instance, batch *mixmessages.Batch) error {

	round := instance.GetRoundManager().GetRound(id.Round(batch.Round.ID))

	if round == nil {
		return errors.New(fmt.Sprintf("Unknown round ID (%v) cannot continue", batch.Round.ID))
	}

	if batch.ForPhase < 0 || batch.ForPhase > int32(phase.NUM_PHASES) {
		return errors.New(fmt.Sprintf("Unknown phase (%v) cannot continue", batch.ForPhase))
	}

	phaseType := phase.Type(uint32(batch.ForPhase))

	p := round.GetPhase(phaseType)

	if !p.ReadyToReceiveData() {
		return errors.New(fmt.Sprintf("Phase %v of round %v is not ready to recieve", p.GetType().String(), round.GetID()))
	}

	instance.GetResourceQueue().UpsertPhase(p)

	for index, messages := range batch.Slots {
		err := p.GetGraph().GetStream().Input(uint32(index), messages)
		if err != nil {
			return err
		}
		p.GetGraph().Send(services.NewChunk(uint32(index), uint32(index+1)))
	}

	return nil

}*/
