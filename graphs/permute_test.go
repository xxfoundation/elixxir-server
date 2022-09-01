////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package graphs

import (
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/shuffle"
	"gitlab.com/elixxir/server/services"
	"gitlab.com/xx_network/crypto/large"
	"runtime"
	"testing"
)

func TestPermute_PrecanPermute(t *testing.T) {
	grp := initPermuteGraphGroup()
	batchSize := uint32(10)
	permutations := make([]uint32, batchSize)

	numPermuted := 3
	ios := make([]PermuteIO, numPermuted)

	for i := 0; i < numPermuted; i++ {
		ios[i] = PermuteIO{
			Input:  grp.NewIntBuffer(batchSize, grp.NewInt(int64(i+1))),
			Output: make([]*cyclic.Int, batchSize),
		}
	}

	shuffle.Shuffle32(&permutations)

	PrecanPermute(permutations, ios...)

	for i := range permutations {
		for _, io := range ios {
			if io.Input.Get(uint32(i)).Cmp(io.Output[permutations[i]]) != 0 {
				t.Errorf("PrecanPermute: Output permutation doesnt match input "+
					"Expected: %v, Received: %v", io.Input.Get(uint32(i)), io.Output[permutations[i]])

			}
		}
	}
}

func TestModifyGraphGeneratorForPermute(t *testing.T) {

	gc := services.NewGraphGenerator(4, uint8(runtime.NumCPU()), 1, 0)

	gcPermute := ModifyGraphGeneratorForPermute(gc)

	if gcPermute.GetOutputSize() != gc.GetOutputSize() {
		t.Errorf("ModifyGraphGeneratorForPermute: Output not copied correctly, "+
			"Expected: %v, Received: %v", gc.GetOutputSize(), gcPermute.GetOutputSize())
	}

	if gcPermute.GetDefaultNumTh() != gc.GetDefaultNumTh() {
		t.Errorf("ModifyGraphGeneratorForPermute: DefaultNumThreads not copied correctly, "+
			"Expected: %v, Received: %v", gc.GetDefaultNumTh(), gcPermute.GetDefaultNumTh())
	}

	if gcPermute.GetMinInputSize() != gc.GetMinInputSize() {
		t.Errorf("ModifyGraphGeneratorForPermute: MinInputSize not copied correctly, "+
			"Expected: %v, Received: %v", gc.GetMinInputSize(), gcPermute.GetMinInputSize())
	}

	// Test that the function on the output is the same as the input
	actualThreshold := float64(gcPermute.GetOutputThreshold())
	expectedThreshold := 1.0
	if expectedThreshold != actualThreshold {
		t.Errorf("ModifyGraphGeneratorForPermute: OutputThreshold not set correctly, "+
			"Expected: %v, Received: %v", expectedThreshold, actualThreshold)
	}
}

func initPermuteGraphGroup() *cyclic.Group {
	primeString := "FFFFFFFFFFFFFFFFC90FDAA22168C234C4C6628B80DC1CD1" +
		"29024E088A67CC74020BBEA63B139B22514A08798E3404DD" +
		"EF9519B3CD3A431B302B0A6DF25F14374FE1356D6D51C245" +
		"E485B576625E7EC6F44C42E9A637ED6B0BFF5CB6F406B7ED" +
		"EE386BFB5A899FA5AE9F24117C4B1FE649286651ECE45B3D" +
		"C2007CB8A163BF0598DA48361C55D39A69163FA8FD24CF5F" +
		"83655D23DCA3AD961C62F356208552BB9ED529077096966D" +
		"670C354E4ABC9804F1746C08CA18217C32905E462E36CE3B" +
		"E39E772C180E86039B2783A2EC07A28FB5C55DF06F4C52C9" +
		"DE2BCBF6955817183995497CEA956AE515D2261898FA0510" +
		"15728E5A8AACAA68FFFFFFFFFFFFFFFF"
	grp := cyclic.NewGroup(large.NewIntFromString(primeString, 16), large.NewInt(2))
	return grp
}
