////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package conf

import "gitlab.com/elixxir/crypto/cyclic"

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

func toGroup(strings map[string]string) *cyclic.Group {
	return nil
}
