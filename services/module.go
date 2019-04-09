////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package services

import (
	"fmt"
	"gitlab.com/elixxir/crypto/cryptops"
	"math"
)

const (
	INPUT_IS_BATCHSIZE = math.MaxUint32
	AUTO_INPUTSIZE     = 0
)

type adapter func(stream Stream, cryptop cryptops.Cryptop, chunk Chunk) error

type Module struct {
	/*Public*/
	// Function which is called by dispatcher and interfaces with the cryptop
	Adapt adapter
	// Cryptographic code which is executed
	Cryptop cryptops.Cryptop

	//Number of slots an input subtends
	InputSize uint32
	//Percent of the batch which must be complete
	StartThreshold float32

	//Name of module. Used for debugging.
	Name string

	//Number of goroutines to execute the adapter and cryptops on
	NumThreads uint32

	/*Private*/
	//Keeps track and controls all threads executing in the cryptop
	state moduleState
	//Contains and controls the input channel
	moduleInput
	//Internal id of module used for tracking
	id uint64
	//Slice of modules whose outputs feed into this modules input
	inputModules []*Module
	//Slice of modules who inputs are fed by this modules output
	outputModules []*Module
	//Tracks inputs to the module and determines when they are ready to be processed
	assignmentList assignmentList

	//Tracks if the module is running
	initialized bool
	//denotes if the module has been used
	used bool
	//denoted if it is a copy
	copy bool
}

//Checks inputs are correct and sets the inputsize if it is set to auto
func (m *Module) checkParameters(minInputSize uint32) {
	if m.NumThreads == 0 {
		panic(fmt.Sprintf("Module %s cannot have zero threads", m.Name))
	}

	if m.InputSize == AUTO_INPUTSIZE {
		m.InputSize = ((m.Cryptop.GetInputSize() + minInputSize - 1) / minInputSize) * minInputSize
	}

	if m.InputSize < minInputSize {
		panic(fmt.Sprintf("Module %s cannot have an input size less than %v", m.Name, minInputSize))
	}
}

//Builds assignments
func (m *Module) buildAssignments(batchsize uint32) {

	m.assignmentList.threshold = threshold(batchsize, m.StartThreshold)

	if m.InputSize == INPUT_IS_BATCHSIZE {
		m.InputSize = batchsize
	}

	numJobs := uint32(batchsize / m.InputSize)

	numInputModules := uint32(len(m.inputModules))
	if numInputModules < 1 {
		numInputModules = 1
	}

	m.assignmentList.maxCount = m.InputSize * numInputModules

	primed := uint32(0)
	m.assignmentList.primed = &primed
	m.assignmentList.assignments = make([]*assignment, numJobs)
	m.assignmentList.completed = new(uint32)
	m.assignmentList.numSlots = m.InputSize

	for j := uint32(0); j < numJobs; j++ {
		m.assignmentList.assignments[j] = newAssignment(uint32(j * m.InputSize))
	}
}

//Get the threshold number
func threshold(batchsize uint32, thresh float32) uint32 {
	if thresh < 0 || thresh > 1 {
		panic(fmt.Sprintf("utput threshold was %v, must be between 0 and 1", thresh))
	}
	return uint32(math.Floor(float64(thresh) * float64(batchsize)))
}

func (m Module) DeepCopy() *Module {
	if m.used == true {
		panic("cannot copy a module which is in use")
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
