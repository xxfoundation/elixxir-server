////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package phase

//The Name of a phase
type Type uint32

const (
	// Precomputation Generation: Initializes all the random values in round
	PRECOMP_GENERATION Type = iota

	// Precomputation Share: Combine partial recipient public cypher keys
	PRECOMP_SHARE

	// Precomputation Decrypt: Adds in first set of encrypted keys
	PRECOMP_DECRYPT

	// Precomputation Decrypt: Adds in second set of encrypted keys and permutes
	PRECOMP_PERMUTE

	// Precomputation Reveal: Generates public key to decrypt keys
	PRECOMP_REVEAL

	// Precomputation Strip: Decrypts the Precomputation
	PRECOMP_STRIP

	// Realtime Decrypt: Removes Transmission Keys and add first Keys
	REAL_DECRYPT

	// Realtime Permute: Permutes Slots and add in second keys
	REAL_PERMUTE

	// Realtime Identify: Uses Precomputation to reveal Recipient, broadcasts
	REAL_IDENTIFY

	// Error: A Fatal Error has occurred, cannot continue
	ERROR
)

// Number of phases
const NUM_PHASES Type = ERROR + 1

//Array used to get the Phase Names for Printing
var typeStrings = [NUM_PHASES]string{"PRECOMP_GENERATION",
	"PRECOMP_SHARE", "PRECOMP_DECRYPT", "PRECOMP_PERMUTE",
	"PRECOMP_REVEAL", "PRECOMP_STRIP", "REAL_DECRYPT", "REAL_PERMUTE",
	"REAL_IDENTIFY",
	"ERROR"}

// Adheres to the Stringer interface to return the name of the phase type
func (p Type) String() string {
	return typeStrings[p]
}
