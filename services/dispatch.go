////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package services

import (
	"fmt"
	"github.com/pkg/errors"
	"strings"
	"time"
)

var InvalidTypeAssert = errors.New("type assert failed")

// AdaptMeasureName is the name used to measure how long the adapt function
// takes in dispatch.
const AdaptMeasureName = "Adapt"

// OutModsMeasureName is the name used to measure how long the outputModules
// processing loop takes in the dispatch function.
const OutModsMeasureName = "Mod"

func dispatch(g *Graph, m *Module, threadID uint64) {

	s := g.stream

	// We measure the adapt and the mod
	atID := fmt.Sprintf("%s%d", AdaptMeasureName, threadID)
	omID := fmt.Sprintf("%s%d", OutModsMeasureName, threadID)

	for chunk, cont := <-m.input; cont; chunk, cont = <-m.input {
		g.Lock()
		g.metrics.Measure(atID)
		g.Unlock()
		err := m.Adapt(s, m.Cryptop, chunk)
		g.Lock()
		g.metrics.Measure(atID)
		g.Unlock()

		if err != nil {
			go g.errorHandler(g.name, m.Name, err)
		}

		g.Lock()
		g.metrics.Measure(omID)
		g.Unlock()
		for _, om := range m.outputModules {
			chunkList, err := om.assignmentList.PrimeOutputs(chunk)
			if err != nil {
				go g.errorHandler(g.name, m.Name, err)
				g.Lock()
				g.metrics.Measure(omID)
				g.Unlock()
				return
			}

			for _, r := range chunkList {
				/*fmt.Printf( "%s sending (%v - %v) to %s \n",
				m.Name, r.begin, r.end, om.Name)*/
				om.input <- r
			}

			fin, err := om.assignmentList.DenoteCompleted(len(chunkList))

			if err != nil {
				go g.errorHandler(g.name, m.Name, err)
				g.Lock()
				g.metrics.Measure(omID)
				g.Unlock()
				return
			}
			if fin {
				om.closeInput()
			}
		}
		g.Lock()
		g.metrics.Measure(omID)
		g.Unlock()

	}
}

// GetMetrics aggregates all the dispatch metrics and returns the total time
// spent inside the adapt function and inside the output modules processing loop
func (g *Graph) GetMetrics() (time.Duration, time.Duration) {
	// Get every event and generate the time deltas
	g.Lock()
	events := g.metrics.GetEvents()
	g.Unlock()
	times := make(map[string]time.Time)
	deltas := make(map[string]time.Duration)
	for _, e := range events {
		lastTime, ok := times[e.Tag]
		if ok {
			deltas[e.Tag] = e.Timestamp.Sub(lastTime)
			delete(times, e.Tag)
		} else {
			times[e.Tag] = e.Timestamp
		}
	}

	// Look at each delta and sum them by type
	var modTime, adaptTime time.Duration
	for k, v := range deltas {
		if strings.Contains(k, AdaptMeasureName) {
			adaptTime += v
		}
		if strings.Contains(k, OutModsMeasureName) {
			modTime += v
		}
	}
	return adaptTime, modTime
}
