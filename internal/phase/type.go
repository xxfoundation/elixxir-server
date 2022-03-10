///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package phase

// type.go contains the type a phase can be in

// Type the Name of a phase
type Type uint32

const (
	// PrecompGeneration initializes all the random values in round
	PrecompGeneration Type = iota

	// PrecompShare combines partial recipient public cypher keys
	// Has verification step: Last node broadcasts resulting public cypher key
	PrecompShare

	// PrecompDecrypt adds in first set of encrypted keys
	PrecompDecrypt

	// PrecompPermute adds in second set of encrypted keys and permutes
	PrecompPermute

	// PrecompReveal generates public key to decrypt keys
	// Has verification step: Last node decrypts and broadcasts precomputation
	PrecompReveal

	// RealDecrypt removes Transmission Keys and add first Keys
	RealDecrypt

	// RealPermute permutes Slots and add in second keys
	// Has verification step: Last node uses precomputation to decrypt
	// recipients, and broadcasts the recipients
	RealPermute

	// Complete phase denotes the round has completed
	Complete

	// PhaseError a Fatal Error has occurred, cannot continue
	PhaseError
)

// NumPhases in a Round
const NumPhases = PhaseError + 1

// Verification text
const Verification = "Verification"

// Array used to get the phase Names for Printing
var typeStrings = [NumPhases]string{"PrecompGeneration",
	"PrecompShare", "PrecompDecrypt", "PrecompPermute",
	"PrecompReveal", "RealDecrypt", "RealPermute",
	"PhaseError"}

// Adheres to the Stringer interface to return the name of the phase type
func (p Type) String() string {
	return typeStrings[p]
}
