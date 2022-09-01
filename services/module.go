////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package services

import (
	"fmt"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/gpumathsgo/cryptops"
	"math"
)

const (
	InputIsBatchSize = math.MaxUint32
	AutoInputSize    = 0
)

// Declare the signature of the Module.Adapt function call
type adapter func(stream Stream, cryptop cryptops.Cryptop, chunk Chunk) error

type Module struct {
	/*Public*/
	// Function which is called by dispatcher which interfaces with and performs the cryptop
	Adapt adapter
	// Cryptographic code which is executed
	Cryptop cryptops.Cryptop

	// Number of slots an input subtends
	InputSize uint32
	// Percent of the batch which must be complete
	StartThreshold float32

	// Name of module. Used for debugging.
	Name string

	// Number of goroutines to execute the adapter and cryptops on
	NumThreads uint8

	/*Private*/
	// Contains and controls the input channel
	moduleInput
	// Internal id of module used for tracking
	id uint64
	// Slice of modules whose outputs feed into this modules input
	inputModules []*Module
	// Slice of modules whose inputs are fed by this modules output
	outputModules []*Module
	// Tracks inputs to the module and determines when they are ready to be processed
	assignmentList assignmentList

	// Tracks if the module is running
	initialized bool
	// denotes if the module has been used
	used bool
	// denoted if this module is a copy
	copy bool
}

//Checks inputs are correct and sets the inputSize if it is set to auto
func (m *Module) checkParameters(minInputSize uint32, defaultNumThreads uint8) {
	if m.NumThreads == AutoNumThreads {
		m.NumThreads = defaultNumThreads
	}

	if m.InputSize == AutoInputSize {
		m.InputSize = ((m.Cryptop.GetInputSize() + minInputSize - 1) / minInputSize) * minInputSize
	}

	if m.InputSize < minInputSize {
		jww.FATAL.Panicf(fmt.Sprintf("Module %s cannot have an input size less"+
			" than %v",
			m.Name, minInputSize))
	}
}

//Builds assignments
func (m *Module) buildAssignments(batchSize uint32) {

	m.assignmentList.threshold = threshold(batchSize, m.StartThreshold)

	if m.InputSize == InputIsBatchSize {
		m.InputSize = batchSize
	}

	if batchSize%m.InputSize != 0 {
		jww.FATAL.Panicf("%v expanded batch size incorrect: "+
			"module input size is not factor; BatchSize: %v, Module Input: %v ",
			m.Name, batchSize, m.InputSize)
	}

	numJobs := batchSize / m.InputSize

	numInputModules := uint32(len(m.inputModules))
	if numInputModules < 1 {
		numInputModules = 1
	}

	m.assignmentList.maxCount = m.InputSize * numInputModules

	waitingIndex := uint32(0)
	waitingAdded := uint32(0)
	m.assignmentList.waitingIndex = &waitingIndex
	m.assignmentList.waitingAdded = &waitingAdded
	m.assignmentList.assignments = make([]*assignment, numJobs)
	m.assignmentList.completed = new(uint32)
	m.assignmentList.numSlots = m.InputSize

	m.assignmentList.waiting = make([]Chunk, numJobs)

	for j := uint32(0); j < numJobs; j++ {
		m.assignmentList.assignments[j] = newAssignment(j * m.InputSize)
	}
}

//Get the threshold number
func threshold(batchSize uint32, thresh float32) uint32 {
	if thresh < 0 || thresh > 1 {
		jww.FATAL.Panicf("output threshold was %v, "+
			"must be between 0 and 1", thresh)
	}
	return uint32(float64(thresh) * float64(batchSize))
}

func (m Module) DeepCopy() *Module {
	if m.used == true {
		jww.FATAL.Panicf("cannot copy a module which is in use")
	}

	mCopy := Module{
		Adapt:          m.Adapt,
		Cryptop:        m.Cryptop,
		NumThreads:     m.NumThreads,
		InputSize:      m.InputSize,
		StartThreshold: m.StartThreshold,
		Name:           m.Name,
	}

	mCopy.copy = true

	return &mCopy
}

func (m Module) GetID() uint64 {
	return m.id
}
