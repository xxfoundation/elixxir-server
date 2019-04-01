package cryptops

type MultiMulSignature func(X, Y, Z []int) []int

var MultiMul MultiMulSignature = func(X, Y, Z []int) []int {
	for i := range Z {
		Z[i] = X[i] * Y[i]
	}
	return Z
}

func (MultiMulSignature) GetFuncName() string {
	return "Mul"
}

func (MultiMulSignature) GetMinSize() uint32 {
	return 4
}
