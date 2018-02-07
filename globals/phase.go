package globals

type Phase uint8

const (

	// Off: An Initialized round which hasn't been started by the master yet
	OFF Phase = iota

	// Precomputation Generation: Initializes all the random values in round
	PRECOMP_GENERATION

	// Precomputation Share: Combine partial recipient public cypher keys
	PRECOMP_SHARE

	// Precomputation Decrypt: Adds in first set of encrypted keys
	PRECOMP_DECRYPT

	// Precomputation Decrypt: Adds in second set of encrypted keys and permutes
	PRECOMP_PERMUTE

	// Precomputation Encrypt: Adds in last set of encrypted keys
	PRECOMP_ENCRYPT

	// Precomputation Reveal: Generates public key to decrypt keys
	PRECOMP_REVEAL

	// Precomputation Strip: Decrypts the Precomputation
	PRECOMP_STRIP

	// Wait: Precomputation has finished but Realtime hasn't started
	WAIT

	// Realtime Decrypt: Removes Transmission Keys and add first Keys
	REAL_DECRYPT

	// Realtime Permute: Permutes Slots and add in second keys
	REAL_PERMUTE

	// Realtime Identify: Uses Precomputation to reveal Recipient, broadcasts
	REAL_IDENTIFY

	// Realtime Encrypt: Add in Reception Keys and Last Keys
	REAL_ENCRYPT

	// Realtime Peel: Uses Precomputation to prepare slots for Reception
	REAL_PEEL

	// Done: Round has been completed
	DONE

	// Error: A Fatal Error has occurred, cannot continue
	ERROR
)

// Number of phases
const NUM_PHASES Phase = ERROR + 1

//Array used to get the Phase Names for Printing
var phaseNames [NUM_PHASES]string

func (p Phase) String() string {

	if phaseNames[0] == "" {
		phaseNames = [...]string{"OFF", "PRECOMP_GENERATION",
			"PRECOMP_SHARE", "PRECOMP_DECRYPT", "PRECOMP_PERMUTE",
			"PRECOMP_ENCRYPT", "PRECOMP_REVEAL", "PRECOMP_STRIP",
			"WAIT", "REAL_DECRYPT", "REAL_PERMUTE",
			"REAL_IDENTIFY", "REAL_ENCRYPT", "REAL_PEEL", "DONE",
			"ERROR"}
	}

	return phaseNames[p]
}
