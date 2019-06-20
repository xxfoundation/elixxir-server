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

// Contains the cyclic group config params
type Groups struct {
	CMix map[string]string `yaml:"cmix"`
	E2E  map[string]string `yaml:"e2e"`
}

func (g Groups) GetCMix() *cyclic.Group {
	return toGroup(g.CMix)
}

func (g Groups) GetE2E() *cyclic.Group {
	return toGroup(g.E2E)
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
// The string representation must be in hex.  It can either
// be preceded by an 0x or not.
func toLargeInt(hexStr string) *large.Int {
	if len(hexStr) > 2 && "0x" == hexStr[:2] {
		return large.NewIntFromString(hexStr[2:], 16)
	}
	return large.NewIntFromString(hexStr, 16)
}
