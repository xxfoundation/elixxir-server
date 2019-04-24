package node

import (
	"errors"
)

var ErrOutsideOfGroup = errors.New("cyclic int is outside of the prescribed group")
var ErrOutsideOfBatch = errors.New("cyclic int is outside of the prescribed batch")
