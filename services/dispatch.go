package services

import (
	"errors"
)

var InvalidTypeAssert = errors.New("type assert failed")

func dispatch(g *Graph, m *Module, threadID uint8) {
	s := g.stream

	done := false

	for !done {

		select {
		case chunk, ok := <-m.input:
			if !ok {

				m.denoteClose(threadID, nil)
				done = true
			} else {
				err := m.Adapt(s, m.Cryptop, chunk)

				if err != nil {
					go g.callback(err)
				}

				for _, om := range m.outputModules {

					chunkList := om.PrimeOutputs(chunk)
					for _, r := range chunkList {
						om.input <- r
					}

					fin := om.DenoteCompleted(len(chunkList))

					if fin {
						// Here the receiver is closing the input, from multiple senders? This is extremely likely to be wrong
						// Although, it seems like closing the channels might be coupled to killing the worker threads somehow?
						// We need to remove that dependency ASAP. Receiving a
						// kill signal should just cause any workers to
						// immediately return at the next opportunity.
						om.closeInput()
					}
				}
			}
		case killNotify := <-m.threads[threadID]:
			m.denoteClose(threadID, killNotify)
			done = true
			for _, om := range m.outputModules {
				om.closeInput()
			}
		}
	}
	//check to ensure all output channels are closed.  Only has an effect on errors/failures.
	if !m.AnyRunning() {
		for _, om := range m.outputModules {
			om.closeInput()
		}
	}
}
