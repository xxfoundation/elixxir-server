package phase

import (
	"fmt"
	"gitlab.com/elixxir/primitives/id"
)

type Fingerprint struct {
	tYpe Type
	round id.Round
}

//Cmp returns true if the fingerprints are the same, false if they are different
func (f Fingerprint) Cmp(f2 Fingerprint) bool {
	return f.round == f2.round && f.tYpe == f2.tYpe
}

//String adheres to the Stringer Interface
func (f Fingerprint) String() string {
	return fmt.Sprintf("phase.Fingerprint{RoundID: %v, Phase: %v}", f.round,
		f.tYpe.String())
}
