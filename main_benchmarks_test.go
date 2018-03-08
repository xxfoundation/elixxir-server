////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package main

import (
	"gitlab.com/privategrity/crypto/cyclic"
	// "gitlab.com/privategrity/server/cryptops/precomputation"
	// "gitlab.com/privategrity/server/cryptops/realtime"
	// "gitlab.com/privategrity/server/globals"
	// "gitlab.com/privategrity/server/services"
	// "strconv"
	"testing"
)

var PRIME = "FFFFFFFFFFFFFFFFC90FDAA22168C234C4C6628B80DC1CD1" +
		"29024E088A67CC74020BBEA63B139B22514A08798E3404DD" +
		"EF9519B3CD3A431B302B0A6DF25F14374FE1356D6D51C245" +
		"E485B576625E7EC6F44C42E9A637ED6B0BFF5CB6F406B7ED" +
		"EE386BFB5A899FA5AE9F24117C4B1FE649286651ECE45B3D" +
		"C2007CB8A163BF0598DA48361C55D39A69163FA8FD24CF5F" +
		"83655D23DCA3AD961C62F356208552BB9ED529077096966D" +
		"670C354E4ABC9804F1746C08CA18217C32905E462E36CE3B" +
		"E39E772C180E86039B2783A2EC07A28FB5C55DF06F4C52C9" +
		"DE2BCBF6955817183995497CEA956AE515D2261898FA0510" +
		"15728E5A8AAAC42DAD33170D04507A33A85521ABDF1CBA64" +
		"ECFB850458DBEF0A8AEA71575D060C7DB3970F85A6E1E4C7" +
		"ABF5AE8CDB0933D71E8C94E04A25619DCEE3D2261AD2EE6B" +
		"F12FFA06D98A0864D87602733EC86A64521F2B18177B200C" +
		"BBE117577A615D6C770988C0BAD946E208E24FA074E5AB31" +
		"43DB5BFCE0FD108E4B82D120A92108011A723C12A787E6D7" +
		"88719A10BDBA5B2699C327186AF4E23C1A946834B6150BDA" +
		"2583E9CA2AD44CE8DBBBC2DB04DE8EF92E8EFC141FBECAA6" +
		"287C59474E6BC05D99B2964FA090C3A2233BA186515BE7ED" +
		"1F612970CEE2D7AFB81BDD762170481CD0069127D5B05AA9" +
		"93B4EA988D8FDDC186FFB7DC90A6C08F4DF435C934063199" +
		"FFFFFFFFFFFFFFFF"


// Tempate function for running the variations of GenerateRounds
func RoundGeneratorBenchmark(nodeCount int, batchSize uint64, b *testing.B) {
	prime := cyclic.NewInt(0)
	prime.SetString(PRIME, 16)

	rng := cyclic.NewRandom(cyclic.NewInt(0), cyclic.NewInt(1000))
	grp := cyclic.NewGroup(prime, cyclic.NewInt(5), cyclic.NewInt(4),
		rng)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GenerateRounds(nodeCount, batchSize, &grp, b)
	}
}

// GenerateRoundsBenchmarkTests
func BenchmarkGenerateRounds_5_1024(b *testing.B) {
	RoundGeneratorBenchmark(5, 1024, b)
}


// Template function for running precomputation
func Precomp(nodeCount int, batchSize uint64, b *testing.B) {
	if testing.Short() {
		cnt := uint64(nodeCount) * batchSize
		if cnt > 512 {
			b.Skip("Skipping test due to short mode flag")
		}
	}
	prime := cyclic.NewInt(0)
	prime.SetString(PRIME, 16)

	rng := cyclic.NewRandom(cyclic.NewInt(0), cyclic.NewInt(1000))
	grp := cyclic.NewGroup(prime, cyclic.NewInt(5), cyclic.NewInt(4),
		rng)
	rounds := GenerateRounds(nodeCount, batchSize, &grp, b)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		MultiNodePrecomp(nodeCount, batchSize, &grp, rounds, b)
	}
}

func BenchmarkPrecomp_1_1(b *testing.B) { Precomp(1, 1, b) }
func BenchmarkPrecomp_1_2(b *testing.B) { Precomp(1, 2, b) }
func BenchmarkPrecomp_1_4(b *testing.B) { Precomp(1, 4, b) }
func BenchmarkPrecomp_1_8(b *testing.B) { Precomp(1, 8, b) }
func BenchmarkPrecomp_1_16(b *testing.B) { Precomp(1, 16, b) }
func BenchmarkPrecomp_1_32(b *testing.B) { Precomp(1, 32, b) }
func BenchmarkPrecomp_1_64(b *testing.B) { Precomp(1, 64, b) }
func BenchmarkPrecomp_1_128(b *testing.B) { Precomp(1, 128, b) }
func BenchmarkPrecomp_1_256(b *testing.B) { Precomp(1, 256, b) }
func BenchmarkPrecomp_1_512(b *testing.B) { Precomp(1, 512, b) }
func BenchmarkPrecomp_1_1024(b *testing.B) { Precomp(1, 1024, b) }

func BenchmarkPrecomp_3_1(b *testing.B) { Precomp(3, 1, b) }
func BenchmarkPrecomp_3_2(b *testing.B) { Precomp(3, 2, b) }
func BenchmarkPrecomp_3_4(b *testing.B) { Precomp(3, 4, b) }
func BenchmarkPrecomp_3_8(b *testing.B) { Precomp(3, 8, b) }
func BenchmarkPrecomp_3_16(b *testing.B) { Precomp(3, 16, b) }
func BenchmarkPrecomp_3_32(b *testing.B) { Precomp(3, 32, b) }
func BenchmarkPrecomp_3_64(b *testing.B) { Precomp(3, 64, b) }
func BenchmarkPrecomp_3_128(b *testing.B) { Precomp(3, 128, b) }
func BenchmarkPrecomp_3_256(b *testing.B) { Precomp(3, 256, b) }
func BenchmarkPrecomp_3_512(b *testing.B) { Precomp(3, 512, b) }
func BenchmarkPrecomp_3_1024(b *testing.B) { Precomp(3, 1024, b) }

func BenchmarkPrecomp_5_1(b *testing.B) { Precomp(5, 1, b) }
func BenchmarkPrecomp_5_2(b *testing.B) { Precomp(5, 2, b) }
func BenchmarkPrecomp_5_4(b *testing.B) { Precomp(5, 4, b) }
func BenchmarkPrecomp_5_8(b *testing.B) { Precomp(5, 8, b) }
func BenchmarkPrecomp_5_16(b *testing.B) { Precomp(5, 16, b) }
func BenchmarkPrecomp_5_32(b *testing.B) { Precomp(5, 32, b) }
func BenchmarkPrecomp_5_64(b *testing.B) { Precomp(5, 64, b) }
func BenchmarkPrecomp_5_128(b *testing.B) { Precomp(5, 128, b) }
func BenchmarkPrecomp_5_256(b *testing.B) { Precomp(5, 256, b) }
func BenchmarkPrecomp_5_512(b *testing.B) { Precomp(5, 512, b) }
func BenchmarkPrecomp_5_1024(b *testing.B) { Precomp(5, 1024, b) }

func BenchmarkPrecomp_10_1(b *testing.B) { Precomp(10, 1, b) }
func BenchmarkPrecomp_10_2(b *testing.B) { Precomp(10, 2, b) }
func BenchmarkPrecomp_10_4(b *testing.B) { Precomp(10, 4, b) }
func BenchmarkPrecomp_10_8(b *testing.B) { Precomp(10, 8, b) }
func BenchmarkPrecomp_10_16(b *testing.B) { Precomp(10, 16, b) }
func BenchmarkPrecomp_10_32(b *testing.B) { Precomp(10, 32, b) }
func BenchmarkPrecomp_10_64(b *testing.B) { Precomp(10, 64, b) }
func BenchmarkPrecomp_10_128(b *testing.B) { Precomp(10, 128, b) }
func BenchmarkPrecomp_10_256(b *testing.B) { Precomp(10, 256, b) }
func BenchmarkPrecomp_10_512(b *testing.B) { Precomp(10, 512, b) }
func BenchmarkPrecomp_10_1024(b *testing.B) { Precomp(10, 1024, b) }

