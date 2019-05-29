////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package conf

import (
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/large"
	"strings"
)

// Contains the cyclic group config params
type Groups struct {
	CMix *cyclic.Group
	E2E  *cyclic.Group
}

// NewGroups creates a groups object from
// a viper config.
// TODO: This is a hack and UnmarshalYAML
// should likely be used, but wasn't working
// with viper.  Perhaps a missing decoder option?
func NewGroups(vip *viper.Viper) Groups {

	cmix := vip.GetStringMapString("groups.cmix")
	e2e := vip.GetStringMapString("groups.e2e")

	return Groups{
		CMix: toGroup(cmix),
		E2E:  toGroup(e2e),
	}
}

// TODO: field names start with a capital by convention
// but perhaps we should override to force a consistent scheme
// See yaml package documentation for more info.
type groups struct {
	Cmix map[string]string
	E2e  map[string]string
}

// UnmarshalYAML defines custom unmarshalling behavior
// such that exported Group structure can contain cyclic Groups
// using the internal Groups struct which contains string mappings
func (Grps *Groups) UnmarshalYAML(unmarshal func(interface{}) error) error {

	grps := groups{}

	err := unmarshal(&grps)

	if err != nil {
		return err
	}

	Grps.CMix = toGroup(grps.Cmix)
	Grps.E2E = toGroup(grps.E2e)

	return nil
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
