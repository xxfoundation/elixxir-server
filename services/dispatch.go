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
	var chunk Chunk

	for !done {
		chunk, done = <-m.input

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
}
