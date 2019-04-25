package phase

import "gitlab.com/elixxir/primitives/id"

type Fingerprint struct {
	phase Type
	round id.Round
}

//Cmp returns true if the fingerprints are the same, false if they are different
func (f Fingerprint) Cmp(f2 Fingerprint) bool {
	return f.round == f2.round && f.phase == f2.phase
}
