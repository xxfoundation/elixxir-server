package cryptops

type SubSignature func(X, Y int) int

var Sub SubSignature = func(X, Y int) int {
	return X - Y
}

func (SubSignature) GetFuncName() string {
	return "Sub"
}

func (SubSignature) GetMinSize() uint32 {
	return 1
}
