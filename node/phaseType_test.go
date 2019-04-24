package node

import "testing"

//Tests that the number of phases and the number of phase strings are the same
func TestPhase(t *testing.T) {
	if len(phaseNames) != int(NUM_PHASES) {
		t.Errorf("Number of phase strings (%v) not equal to number of phases (%v)",
			len(phaseNames), int(NUM_PHASES))
	}
}

//Tests that the correct phases are returned
func TestPhase_String(t *testing.T) {
	for i := PhaseType(0); i < NUM_PHASES; i++ {
		if i.String() != phaseNames[i] {
			t.Errorf("Phase.String does not outpur the correct result for phase %s",
				phaseNames[i])
		}
	}
}
