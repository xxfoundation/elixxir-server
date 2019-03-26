package globals

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/crypto/cyclic"
)

var grp *cyclic.Group

// Allows a global group to be set once.
func SetGroup(g *cyclic.Group) {
	if grp != nil {
		jww.CRITICAL.Panicf("Cannot set the core group twice")
	}

	grp = g
}

// Retrieves set global group.
func GetGroup() *cyclic.Group {
	return grp
}
