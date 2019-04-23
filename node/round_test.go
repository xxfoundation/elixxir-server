////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package node

import (
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/large"
	"math/rand"
	"reflect"
	"testing"
)

var pString = "9DB6FB5951B66BB6FE1E140F1D2CE5502374161FD6538DF1648218642F0B5C48" +
	"C8F7A41AADFA187324B87674FA1822B00F1ECF8136943D7C55757264E5A1A44F" +
	"FE012E9936E00C1D3E9310B01C7D179805D3058B2A9F4BB6F9716BFE6117C6B5" +
	"B3CC4D9BE341104AD4A80AD6C94E005F4B993E14F091EB51743BF33050C38DE2" +
	"35567E1B34C3D6A5C0CEAA1A0F368213C3D19843D0B4B09DCB9FC72D39C8DE41" +
	"F1BF14D4BB4563CA28371621CAD3324B6A2D392145BEBFAC748805236F5CA2FE" +
	"92B871CD8F9C36D3292B5509CA8CAA77A2ADFC7BFD77DDA6F71125A7456FEA15" +
	"3E433256A2261C6A06ED3693797E7995FAD5AABBCFBE3EDA2741E375404AE25B"

var gString = "5C7FF6B06F8F143FE8288433493E4769C4D988ACE5BE25A0E24809670716C613" +
	"D7B0CEE6932F8FAA7C44D2CB24523DA53FBE4F6EC3595892D1AA58C4328A06C4" +
	"6A15662E7EAA703A1DECF8BBB2D05DBE2EB956C142A338661D10461C0D135472" +
	"085057F3494309FFA73C611F78B32ADBB5740C361C9F35BE90997DB2014E2EF5" +
	"AA61782F52ABEB8BD6432C4DD097BC5423B285DAFB60DC364E8161F4A2A35ACA" +
	"3A10B1C4D203CC76A470A33AFDCBDD92959859ABD8B56E1725252D78EAC66E71" +
	"BA9AE3F1DD2487199874393CD4D832186800654760E1E34C09E4D155179F9EC0" +
	"DC4473F996BDCE6EED1CABED8B6F116F7AD9CF505DF0F998E34AB27514B0FFE7"

var qString = "F2C3119374CE76C9356990B465374A17F23F9ED35089BD969F61C6DDE9998C1F"

var p = large.NewIntFromString(pString, 16)
var g = large.NewIntFromString(gString, 16)
var q = large.NewIntFromString(qString, 16)

var grp = cyclic.NewGroup(p, g, q)

func TestNewRound(t *testing.T) {

	rng := rand.New(rand.NewSource(42))

	tests := 30

	for i := 0; i < tests; i++ {
		batchSize := rng.Uint32() % 1000
		expandedBatchSize := uint32(float64(batchSize) * (float64(rng.Uint32()%1000) / 100.00))

		rID := RoundID(33)

		r := NewRound(grp, rID, batchSize, expandedBatchSize)

		if r.batchSize != batchSize {
			t.Errorf("New RoundBuffer: Batch Size not stored correctly, "+
				"Expected %v, Recieved: %v", batchSize, r.batchSize)
		}

		if r.expandedBatchSize != expandedBatchSize {
			t.Errorf("New RoundBuffer: Expanded Batch Size not stored correctly, "+
				"Expected %v, Recieved: %v", expandedBatchSize, r.expandedBatchSize)
		}

		if r.id == rID {
			t.Errorf("New RoundBuffer: ID not set correctly, "+
				"Expected %v, Recieved: %v", rID, r.id)
		}

		if !reflect.DeepEqual(r.Grp.GetFingerprint(), grp.GetFingerprint()) {
			t.Errorf("New RoundBuffer: Group Fingerprints do not match")
		}

		defaultInt := grp.NewInt(1)

		checkIntBuffer(r.R, expandedBatchSize, "round.R", defaultInt, t)
		checkIntBuffer(r.S, expandedBatchSize, "round.S", defaultInt, t)
		checkIntBuffer(r.U, expandedBatchSize, "round.U", defaultInt, t)
		checkIntBuffer(r.V, expandedBatchSize, "round.V", defaultInt, t)

		checkIntBuffer(r.Y_R, expandedBatchSize, "round.R", defaultInt, t)
		checkIntBuffer(r.Y_S, expandedBatchSize, "round.S", defaultInt, t)
		checkIntBuffer(r.Y_T, expandedBatchSize, "round.T", defaultInt, t)
		checkIntBuffer(r.Y_U, expandedBatchSize, "round.U", defaultInt, t)
		checkIntBuffer(r.Y_V, expandedBatchSize, "round.V", defaultInt, t)

		checkIntBuffer(r.MessagePrecomputation, expandedBatchSize, "round.MessagePrecomputation", defaultInt, t)
		checkIntBuffer(r.ADPrecomputation, expandedBatchSize, "round.ADPrecomputation", defaultInt, t)

		if r.CypherPublicKey.Cmp(grp.NewMaxInt()) != 0 {
			t.Errorf("New RoundBuffer: Cypher Public Key not initlized correctly")
		}

		if r.Z.Cmp(grp.NewMaxInt()) != 0 {
			t.Errorf("New RoundBuffer: Z not initlized correctly")
		}

		for itr, p := range r.Permutations {
			if p != uint32(itr) {
				t.Errorf("New RoundBuffer: Permutation on index %v not pointing to itself, pointing to %v",
					itr, p)
			}
		}
	}
}

func checkIntBuffer(ib *cyclic.IntBuffer, expandedBatchSize uint32, source string, defaultInt *cyclic.Int, t *testing.T) {
	if ib.Len() != int(expandedBatchSize) {
		t.Errorf("New RoundBuffer: Length of intBuffer %s not correct, "+
			"Expected %v, Recieved: %v", source, expandedBatchSize, ib.Len())
	}

	numBad := 0
	for i := uint32(0); i < expandedBatchSize; i++ {
		ci := ib.Get(i)
		if ci.Cmp(defaultInt) != 0 {
			numBad++
		}
	}

	if numBad != 0 {
		t.Errorf("New RoundBuffer: Ints in %v/%v intBuffer %s intilized incorrectly",
			numBad, expandedBatchSize, source)
	}
}

//Tests getbatchsize and getextendedbatchsize
func TestRound_Get(t *testing.T) {
	rng := rand.New(rand.NewSource(42))

	tests := 30

	for i := 0; i < tests; i++ {
		batchSize := rng.Uint32() % 1000
		expandedBatchSize := uint32(float64(batchSize) * (float64(rng.Uint32()%1000) / 100.00))

		rID := RoundID(33)

		r := NewRound(grp, rID, batchSize, expandedBatchSize)

		if r.GetBatchSize() != batchSize {
			t.Errorf("RoundBuffer.GetBatchSize: Batch Size not correct, "+
				"Expected %v, Recieved: %v", batchSize, r.GetBatchSize())
		}

		if r.GetExpandedBatchSize() != expandedBatchSize {
			t.Errorf("New RoundBuffer: Expanded Batch Size not correct, "+
				"Expected %v, Recieved: %v", expandedBatchSize, r.GetExpandedBatchSize())
		}
	}
}
