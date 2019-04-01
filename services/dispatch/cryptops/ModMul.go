package cryptops

import "math"

type ModMulSignature func(X, Y, P int) int

var ModMul ModMulSignature = func(X, Y, P int) int {
	return int(math.Abs(float64(X*Y))) % P
}

func (ModMulSignature) GetFuncName() string {
	return "ModMul"
}

func (ModMulSignature) GetMinSize() uint32 {
	return 1
}
