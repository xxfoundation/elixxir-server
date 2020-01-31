package services

import (
	"gitlab.com/elixxir/crypto/cryptops"
	"gitlab.com/elixxir/gpumaths"
	"runtime"
	"testing"
)

// Compare modular exponentiation results with Exp kernel with those doing modular exponentiation on the CPU to show they're both the same
var ModuleExpGPU = Module{
	Adapt:          nil,
	Cryptop:        gpumaths.ExpChunk,
	InputSize:      0,
	StartThreshold: 0,
	Name:           "ExpGPU",
	NumThreads:     2,
}

var ModuleExpCPU = Module{
	Adapt:          nil,
	Cryptop:        cryptops.Exp,
	InputSize:      0,
	StartThreshold: 0,
	Name:           "ExpCPU",
	NumThreads:     uint8(runtime.NumCPU()),
}

func TestCGC(t *testing.T) {

}
