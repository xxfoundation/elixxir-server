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
	PrecompGeneration Type = iota

	// Precomputation Share: Combine partial recipient public cypher keys
	PrecompShare

	// Precomputation Decrypt: Adds in first set of encrypted keys
	PrecompDecrypt

	// Precomputation Decrypt: Adds in second set of encrypted keys and permutes
	PrecompPermute

	// Precomputation Reveal: Generates public key to decrypt keys
	PrecompReveal

	// Precomputation Strip: Decrypts the Precomputation
	PrecompStrip

	// Realtime Decrypt: Removes Transmission Keys and add first Keys
	RealDecrypt

	// Realtime Permute: Permutes Slots and add in second keys
	RealPermute

	// Realtime Identify: Uses Precomputation to reveal Recipient, broadcasts
	RealIdentify

	// Error: A Fatal Error has occurred, cannot continue
	PhaseError
)

// Number of phases
const NUM_PHASES Type = PhaseError + 1

//Array used to get the Phase Names for Printing
var typeStrings = [NUM_PHASES]string{"PrecompGeneration",
	"PrecompShare", "PrecompDecrypt", "PrecompPermute",
	"PrecompReveal", "PrecompStrip", "RealDecrypt", "RealPermute",
	"RealIdentify",
	"PhaseError"}

// Adheres to the Stringer interface to return the name of the phase type
func (p Type) String() string {
	return typeStrings[p]
}
