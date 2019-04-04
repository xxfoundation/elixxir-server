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

					chunkList, numComplete := om.PrimeOutputs(chunk)
					for _, r := range chunkList {
						om.input <- r
					}

					fin := om.DenoteCompleted(numComplete)

					if fin {
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
