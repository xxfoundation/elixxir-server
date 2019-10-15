package conf

import "runtime"

var ExpectedGraphGen = GraphGen{
	minInputSize:    4,
	defaultNumTh:    uint8(runtime.NumCPU()),
	outputSize:      4,
	outputThreshold: 0.0,
}