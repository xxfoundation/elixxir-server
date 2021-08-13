///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package round

// bufferFields.go contains the round.Buffer's InitCryptoFields method

import (
	"git.xx.network/elixxir/crypto/cyclic"
	"git.xx.network/elixxir/crypto/shuffle"
)

// InitCryptoFields sets the private key
// and shuffles the batch up to the batch size
// It keeps the remainder of the batch as it was
func (r *Buffer) InitCryptoFields(g *cyclic.Group) {

	// Set private key using small coprime inverse
	// with 256 bits
	bits := uint32(256)
	g.FindSmallCoprimeInverse(r.Z, bits)

	// Permute up to the batch size
	if r.batchSize <= r.expandedBatchSize {
		batchSizePerm := r.Permutations[:r.batchSize]
		shuffle.Shuffle32(&batchSizePerm)
	}

}
