package phase

import "testing"

// Proves that Cmp checks all known fields of Fingerprint
func TestFingerprint_Cmp(t *testing.T) {
	// f1 should only be equal to f4
	f1 := Fingerprint{
		tYpe: REAL_DECRYPT,
		round: 20,
	}
	f2 := Fingerprint{
		tYpe: REAL_IDENTIFY,
		round: 20,
	}
	f3 := Fingerprint{
		tYpe: REAL_DECRYPT,
		round: 21,
	}
	f4 := Fingerprint{
		tYpe: REAL_DECRYPT,
		round: 20,
	}

	if f1.Cmp(f2) {
		t.Error("f1 should not be equal to f2")
	}
	if f1.Cmp(f3) {
		t.Error("f1 should not be equal to f3")
	}
	if !f1.Cmp(f4) {
		t.Error("f1 should be equal to f4")
	}
}

func TestFingerprint_String(t *testing.T) {
	fingerprint := Fingerprint{
		tYpe: PRECOMP_PERMUTE,
		round: 8,
	}
	fingerprintString := fingerprint.String()
	expected := "phase.Fingerprint{RoundID: 8, Phase: PRECOMP_PERMUTE}"
	if expected != fingerprintString {
		t.Error("Fingerprint string differed from expected. Expected %v, "+
			"got %v", expected, fingerprintString)
	}
}
