////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

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

				m.state.denoteClose(threadID, nil)
				done = true
			} else {
				err := m.Adapt(s, m.Cryptop, chunk)

				if err != nil {
					go g.generator.errorHandler(g.name, m.Name, err)
				}

				for _, om := range m.outputModules {
					chunkList, err := om.assignmentList.PrimeOutputs(chunk)
					if err != nil {
						go g.generator.errorHandler(g.name, m.Name, err)
						return
					}

					for _, r := range chunkList {
						//fmt.Printf( "%s sending (%v - %v) to %s \n",
						//	m.Name, r.begin, r.end, om.Name)
						om.input <- r
					}

					fin, err := om.assignmentList.DenoteCompleted(len(chunkList))

					if err != nil {
						go g.generator.errorHandler(g.name, m.Name, err)
						return
					}

					if fin {
						om.closeInput()
					}
				}
			}
		case killNotify := <-m.state.threads[threadID]:
			m.state.denoteClose(threadID, killNotify)
			done = true
			for _, om := range m.outputModules {
				om.closeInput()
			}
		}
	}
	//check to ensure all output channels are closed.  Only has an effect on errors/failures.
	if !m.state.AnyRunning() {
		for _, om := range m.outputModules {
			om.closeInput()
		}
	}
}
