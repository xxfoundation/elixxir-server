////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package round

import (
	"math/rand"
	"testing"
)

// Tests that after calling InitCryptoFields that
// the Z value is initialized and permutations are
// shuffled up to and including batchSize and remain
// ordered from batchSize+1 to expandedBatchSize
func TestInitCryptoFields_ExpBatchSizeGreaterThanBatchSize(t *testing.T) {
	rng := rand.New(rand.NewSource(42))

	tests := 30

	for i := 0; i < tests; i++ {

		batchSize := rng.Uint32()%1000 + 100

		expandedBatchSize := uint32(float64(batchSize) * (1.0 + float64(rng.Uint32()%100)/100.00))

		r := NewBuffer(grp, batchSize, expandedBatchSize)

		if r.GetBatchSize() != batchSize {
			t.Errorf("NewBuffer: Batch Size not stored correctly, "+
				"Expected %v, Received: %v", batchSize, r.batchSize)
		}

		if r.GetExpandedBatchSize() != expandedBatchSize {
			t.Errorf("NewBuffer: Expanded Batch Size not stored correctly, "+
				"Expected %v, Received: %v", expandedBatchSize, r.expandedBatchSize)
		}

		if r.Z.Cmp(grp.NewMaxInt()) != 0 {
			t.Errorf("NewBuffer: Z not initlized correctly")
		}

		for itr, p := range r.Permutations {
			if p != uint32(itr) {
				t.Errorf("New RoundBuffer: Permutation on index %v not pointing to itself, pointing to %v",
					itr, p)
			}
		}

		// Init batch wide keys
		r.Z = grp.NewIntFromUInt(rng.Uint64())
		r.InitCryptoFields(grp)

		bits := uint32(256)
		expectedZ := grp.FindSmallCoprimeInverse(r.Z, bits)

		if r.Z.Cmp(expectedZ) != 0 {
			t.Errorf("Init batch wide keys: Z not set to correct value")
		}

		// Ensure we have shuffled up to batch size
		sumPerm := uint32(0)
		numEqual := uint32(0)

		for itr, p := range r.Permutations[:batchSize] {
			if itr == int(p) {
				numEqual++
			}
			sumPerm += p
		}

		if numEqual > uint32(0.1*float32(r.GetBatchSize())) {
			t.Errorf("Init batch wide keys: Not sufficiently shuffled")
		}

		if sumPerm != (batchSize)*(batchSize-1)/2 {
			t.Errorf("Init batch wide keys: Mismatch in summing permutations")
		}

		// ensure every index after batch size points to itself
		for itr, p := range r.Permutations[batchSize:] {
			index := itr + int(batchSize)
			if p != uint32(index) {
				t.Errorf("Init batch wide keys: Permutation on index %v not pointing to itself, pointing to %v",
					itr, p)
			}
		}

	}
}

// Tests that after calling InitCryptoFields that
// the Z value is initialized and permutations are
// shuffled up to and including expanded batch size
func TestInitCryptoFields_ExpandedBatchSizeEqBatchSize(t *testing.T) {
	rng := rand.New(rand.NewSource(42))

	tests := 30

	for i := 0; i < tests; i++ {

		batchSize := rng.Uint32()%1000 + 100

		expandedBatchSize := batchSize

		r := NewBuffer(grp, batchSize, expandedBatchSize)

		if r.GetBatchSize() != batchSize {
			t.Errorf("NewBuffer: Batch Size not stored correctly, "+
				"Expected %v, Received: %v", batchSize, r.batchSize)
		}

		if r.GetExpandedBatchSize() != expandedBatchSize {
			t.Errorf("NewBuffer: Expanded Batch Size not stored correctly, "+
				"Expected %v, Received: %v", expandedBatchSize, r.expandedBatchSize)
		}

		if r.Z.Cmp(grp.NewMaxInt()) != 0 {
			t.Errorf("NewBuffer: Z not initlized correctly")
		}

		for itr, p := range r.Permutations {
			if p != uint32(itr) {
				t.Errorf("New RoundBuffer: Permutation on index %v not pointing to itself, pointing to %v",
					itr, p)
			}
		}

		// Init batch wide keys
		r.Z = grp.NewIntFromUInt(rng.Uint64())
		r.InitCryptoFields(grp)

		bits := uint32(256)
		expectedZ := grp.FindSmallCoprimeInverse(r.Z, bits)

		if r.Z.Cmp(expectedZ) != 0 {
			t.Errorf("Init batch wide keys: Z not set to correct value")
		}

		// Ensure we have shuffled up to expanded batch size
		sumPerm := uint32(0)
		numEqual := uint32(0)

		for itr, p := range r.Permutations[:expandedBatchSize] {
			if itr == int(p) {
				numEqual++
			}
			sumPerm += p
		}

		if numEqual > uint32(0.1*float32(r.GetBatchSize())) {
			t.Errorf("Init batch wide keys: Not sufficiently shuffled")
		}

		if sumPerm != (batchSize)*(batchSize-1)/2 {
			t.Errorf("Init batch wide keys: Mismatch in summing permutations")
		}

	}
}
