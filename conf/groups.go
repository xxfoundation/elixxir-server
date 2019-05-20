////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package conf

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/large"
	"strings"
)

type Groups interface {
	GetCMix() *cyclic.Group
	GetE2E() *cyclic.Group
}

type groupsImpl struct {
	cMix *cyclic.Group
	e2e  *cyclic.Group
}

func NewGroups(cMix, e2e map[string]string) Groups {
	return groupsImpl{
		cMix: toGroup(cMix),
		e2e:  toGroup(e2e),
	}
}

func (grps groupsImpl) GetCMix() *cyclic.Group {
	return grps.cMix
}

func (grps groupsImpl) GetE2E() *cyclic.Group {
	return grps.e2e
}

// toGroup takes a group represented by a map of string to string
// and uses the prime, small prime and generator to  created
// and returns a a cyclic group object.
func toGroup(grp map[string]string) *cyclic.Group {
	pStr, pOk := grp["prime"]
	qStr, qOk := grp["smallprime"]
	gStr, gOk := grp["generator"]

	if !gOk || !qOk || !pOk {
		jww.FATAL.Panicf("Invalid Group Config "+
			"(prime: %v, smallPrime: %v, generator: %v",
			pOk, qOk, gOk)
	}

	// TODO: Is there any error checking we should do here? If so, what?
	p := toLargeInt(strings.ReplaceAll(pStr, " ", ""))
	q := toLargeInt(strings.ReplaceAll(qStr, " ", ""))
	g := toLargeInt(strings.ReplaceAll(gStr, " ", ""))

	return cyclic.NewGroup(p, g, q)
}

// toLargeInt takes in a string representation of a large int.
// If the first 2 bytes are '0x' it parses a base 16 number,
// otherwise it parses a base 10 and returns the result.
func toLargeInt(str string) *large.Int {
	if len(str) > 2 && "0x" == str[:2] {
		return large.NewIntFromString(str[2:], 16)
	}
	return large.NewIntFromString(str, 10)
}
