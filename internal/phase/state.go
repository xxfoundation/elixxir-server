///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package phase

// state.go contains the states for a phase

// State a phase is in
type State uint32

const (
	// Initialized Data structures for the phase have been created but it is not ready to run
	Initialized State = iota
	// Active is current phase to be run by the round
	Active
	// Computed graph has computed the result but the phase had not completed
	Computed
	// Verified phase is finished, all required tasks are completed
	Verified
	// NumStates end of const block item: holds number of constants
	NumStates
)

// Array used to get the phase Names for Printing
var stateStrings = []string{"Initialized",
	"Active", "Computed", "Verified"}

// Adheres to the Stringer interface to return the name of the phase type
func (s State) String() string {
	return stateStrings[s]
}
