////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package phase

//The state a phase is in
type State uint32

const (
	// Initialized: Data structures for the phase have been created but it is not ready to run
	Initialized State = iota
	// Active: Current phase to be run by the round
	Active
	// Computed: graph has computed the result but the phase had not completed
	Computed
	// Verified: phase is finished, all required tasks are completed
	Verified
	// End of const block item: holds number of constants
	NumStates
)

// Array used to get the phase Names for Printing
var stateStrings = []string{"Initialized",
	"Active", "Computed", "Verified"}

// Adheres to the Stringer interface to return the name of the phase type
func (s State) String() string {
	return stateStrings[s]
}
