////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package services

import (
	"fmt"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"strings"
	"time"
)

var (
	InvalidTypeAssert = errors.New("type assert failed")
	InvalidMAC        = "User could not be validated"
	SecretNotFound    = "Could not find secret"
	timeoutDuration   = 2 * time.Minute
)

// AdaptMeasureName is the name used to measure how long the adapt function
// takes in dispatch.
const AdaptMeasureName = "Adapt"

// OutModsMeasureName is the name used to measure how long the outputModules
// processing loop takes in the dispatch function.
const OutModsMeasureName = "Mod"

// dispatch runs a Module while taking measurements for the Graph
// and forwards the output of this Module to its output Modules
func dispatch(g *Graph, m *Module, threadID uint64) {
	s := g.stream

	// We measure the adapt function and the mod
	atID := fmt.Sprintf("%s%d", AdaptMeasureName, threadID)
	omID := fmt.Sprintf("%s%d", OutModsMeasureName, threadID)

	var chunk Chunk
	var ok bool
	timeout := time.NewTimer(timeoutDuration)
	keepLooping := true
	for keepLooping {
		select {
		// Time out that channel read in the loop to prevent it getting stuck
		case <-timeout.C:
			keepLooping = false
			jww.WARN.Printf("Graph %v in module %v timed out thread %v", g.GetName(), m.Name, threadID)
		case chunk, ok = <-m.input:
			if ok {
				g.Lock()
				g.metrics.Measure(atID)
				g.Unlock()
				// Run the Module for each chunk
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

					// Send output chunks of this Module to inputs of the output Modules
					for _, r := range chunkList {
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
				timeout.Reset(timeoutDuration)
			} else {
				// normal loop exit
				keepLooping = false
			}
		}
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

	// NOTE: Metrics are ordered, but interleaved with each other.
	// The following unrolls that, so e.g.:
	//   Metric1, Metric2, Metric1, Metric1, Metric2
	// Gets converted to: Delta(Metric1, Metric1), Delta(Metric2, Metric2)
	//                    Delta(Metric1, Metric1), ...
	// and so on, orderd by matching tags in the list.
	for _, e := range events {
		lastTime, ok := times[e.Tag]
		if ok { // lastTime condition
			// If the tag exists, then we had a "previous" event
			// of the same tag. Calculate the delta between
			// this one and the last one.
			deltas[e.Tag] = e.Timestamp.Sub(lastTime)
			// Delete the tag, so we can detect the "startTime"
			// condition.
			delete(times, e.Tag)
		} else { // startTime condition
			times[e.Tag] = e.Timestamp
		}
	}

	// Look at each delta and sum them by type. This collapses each
	// thread's time spent into a single metric for that thread.
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
