////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package services

import "fmt"

// Should probably add more params to this like block ID, worker thread ID, etc
type ErrorCallback func(err error)

type GraphGenerator struct {
	minInputSize    uint32
	errorHandler    ErrorCallback
	defaultNumTh    uint8
	outputSize      uint32
	outputThreshold float32
}

func NewGraphGenerator(minInputSize uint32, errorHandler ErrorCallback, defaultNumTh uint8, outputSize uint32, outputThreshold float32) GraphGenerator {
	if defaultNumTh > MAX_THREADS {
		panic(fmt.Sprintf("Max threads per module is 64, cannot default to %v threads", defaultNumTh))
	}
	if defaultNumTh == 0 {
		panic("Cannot default to zero threads")
	}

	if minInputSize == 0 {
		panic("Minimum input size must be greater than zero")
	}

	if outputSize == 0 {
		panic("OutputSize must be at least 1")
	}

	if outputThreshold < 0.0 || outputSize > 1.0 {
		panic("Output Threshold must be between 0.0 and 1.0")
	}

	return GraphGenerator{
		minInputSize:    minInputSize,
		errorHandler:    errorHandler,
		defaultNumTh:    defaultNumTh,
		outputSize:      outputSize,
		outputThreshold: outputThreshold,
	}
}

func (gc GraphGenerator) GetMinInputSize() uint32 {
	return gc.minInputSize
}

func (gc GraphGenerator) GetDefaultNumTh() uint8 {
	return gc.defaultNumTh
}

func (gc GraphGenerator) NewGraph(name string, stream Stream) *Graph {

	var g Graph
	g.generator = gc
	g.modules = make(map[uint64]*Module)
	g.idCount = 0
	g.batchSize = 0
	g.expandBatchSize = 0

	g.name = name

	g.built = false
	g.linked = false

	g.stream = stream

	g.sentInputs = new(uint32)

	return &g
}
