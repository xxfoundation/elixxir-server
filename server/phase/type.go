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
	// Has verification step: Last node broadcasts resulting public cypher key
	PrecompShare

	// Precomputation Decrypt: Adds in first set of encrypted keys
	PrecompDecrypt

	// Precomputation Decrypt: Adds in second set of encrypted keys and permutes
	PrecompPermute

	// Precomputation Reveal: Generates public key to decrypt keys
	// Has verification step: Last node decrypts and broadcasts precomputation
	PrecompReveal

	// Realtime Decrypt: Removes Transmission Keys and add first Keys
	RealDecrypt

	// Realtime Permute: Permutes Slots and add in second keys
	// Has verification step: Last node uses precomputation to decrypt
	// recipients, and broadcasts the recipients
	RealPermute

	// Complete phase denotes the round has completed
	Complete

	// Error: A Fatal Error has occurred, cannot continue
	PhaseError
)

// Number of phases
const NUM_PHASES Type = PhaseError + 1

//Verification text
const Verification = "Verification"

//Array used to get the phase Names for Printing
var typeStrings = [NUM_PHASES]string{"PrecompGeneration",
	"PrecompShare", "PrecompDecrypt", "PrecompPermute",
	"PrecompReveal", "RealDecrypt", "RealPermute",
	"PhaseError"}

// Adheres to the Stringer interface to return the name of the phase type
func (p Type) String() string {
	return typeStrings[p]
}
