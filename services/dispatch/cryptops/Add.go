package cryptops

type AddSignature func(X, Y int) int

var Add AddSignature = func(X, Y int) int {
	return X + Y
}

func (AddSignature) GetFuncName() string {
	return "Add"
}

func (AddSignature) GetMinSize() uint32 {
	return 1
}
