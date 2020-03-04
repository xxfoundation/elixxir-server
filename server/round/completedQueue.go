package round

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/server/phase"
	"gitlab.com/elixxir/server/services"
)

type CompletedQueue chan *CompletedRound

func (cq CompletedQueue) SendCompletedBatchQueue(cr *CompletedRound) {
	select {
	case cq <- cr:
	default:
		jww.ERROR.Printf("Completed batch queue full, " +
			"batch dropped. Check Gateway")

	}
}

type CompletedRound struct {
	RoundID    id.Round
	Receiver   chan services.Chunk
	GetMessage phase.GetMessage
}

func (cr *CompletedRound) GetChunk() (services.Chunk, bool) {
	chunk, ok := <-cr.Receiver
	return chunk, ok
}
