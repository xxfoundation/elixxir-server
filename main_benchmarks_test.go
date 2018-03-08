////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package main

import (
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/server/globals"
	"gitlab.com/privategrity/server/cryptops/realtime"
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
		if cnt > 256 {
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

func CopyRounds(nodeCount int, r []*globals.Round) ([]*globals.Round) {
	tmp := make([]*globals.Round, nodeCount)
	for i := 0; i < nodeCount; i++ {
		tmp[i] = globals.NewRound(r[i].BatchSize)
		if (i + 1) == nodeCount {
			globals.InitLastNode(tmp[i])
		}
		for j := uint64(0); j < r[i].BatchSize; j++ {
			tmp[i].R[j].Set(r[i].R[j])
			tmp[i].S[j].Set(r[i].S[j])
			tmp[i].T[j].Set(r[i].T[j])
			tmp[i].V[j].Set(r[i].V[j])
			tmp[i].U[j].Set(r[i].U[j])
			tmp[i].R_INV[j].Set(r[i].R_INV[j])
			tmp[i].S_INV[j].Set(r[i].S_INV[j])
			tmp[i].T_INV[j].Set(r[i].T_INV[j])
			tmp[i].V_INV[j].Set(r[i].V_INV[j])
			tmp[i].U_INV[j].Set(r[i].U_INV[j])
			tmp[i].Y_R[j].Set(r[i].Y_R[j])
			tmp[i].Y_S[j].Set(r[i].Y_S[j])
			tmp[i].Y_T[j].Set(r[i].Y_T[j])
			tmp[i].Y_V[j].Set(r[i].Y_V[j])
			tmp[i].Y_U[j].Set(r[i].Y_U[j])
			tmp[i].Permutations[j] = r[i].Permutations[j]
			if (i + 1) == nodeCount {
				tmp[i].LastNode.MessagePrecomputation[j].Set(
					r[i].LastNode.MessagePrecomputation[j])
				tmp[i].LastNode.RecipientPrecomputation[j].Set(
					r[i].LastNode.RecipientPrecomputation[j])
				tmp[i].LastNode.RoundMessagePrivateKey[j].Set(
					r[i].LastNode.RoundMessagePrivateKey[j])
				tmp[i].LastNode.RoundRecipientPrivateKey[j].Set(
					r[i].LastNode.RoundRecipientPrivateKey[j])
				tmp[i].LastNode.RecipientCypherText[j].Set(
					r[i].LastNode.RecipientCypherText[j])
				tmp[i].LastNode.EncryptedRecipientPrecomputation[j].Set(
					r[i].LastNode.EncryptedRecipientPrecomputation[j])
				tmp[i].LastNode.EncryptedMessagePrecomputation[j].Set(
					r[i].LastNode.EncryptedMessagePrecomputation[j])
				tmp[i].LastNode.EncryptedMessage[j].Set(
					r[i].LastNode.EncryptedMessage[j])
			}
		}
		tmp[i].CypherPublicKey.Set(r[i].CypherPublicKey)
		tmp[i].Z.Set(r[i].Z)
	}
	return tmp
}

func GenerateIOMessages(nodeCount int, batchSize uint64,
	rounds []*globals.Round) ([]realtime.RealtimeSlot, []realtime.RealtimeSlot) {
	inputMsgs := make([]realtime.RealtimeSlot, batchSize)
	outputMsgs := make([]realtime.RealtimeSlot, batchSize)
	for i := uint64(0); i < batchSize; i++ {
		inputMsgs[i] = realtime.RealtimeSlot{
			Slot:               i,
			CurrentID:          i + 1,
			Message:            cyclic.NewInt((42 + int64(i)) % 101), // Meaning of Life
			EncryptedRecipient: cyclic.NewInt((1 + int64(i)) % 101),
			CurrentKey:         cyclic.NewInt(1),
		}
		outputMsgs[i] = realtime.RealtimeSlot{
			Slot:      i,
			CurrentID: (i + 1) % 101,
			Message:   cyclic.NewInt((42 + int64(i)) % 101), // Meaning of Life
		}
	}
	for i := 0; i < nodeCount; i++ {
		// Now apply  permutations list to outputMsgs
		newOutMsgs := make([]realtime.RealtimeSlot, batchSize)
		for j := uint64(0); j < batchSize; j++ {
			newOutMsgs[rounds[i].Permutations[j]] = outputMsgs[j]
		}

		copy(outputMsgs, newOutMsgs)
		outputMsgs = newOutMsgs
	}

	return inputMsgs, outputMsgs
}

// Benchmarks for realtime
func Realtime(nodeCount int, batchSize uint64, b *testing.B) {
	if testing.Short() {
		cnt := uint64(nodeCount) * batchSize
		if cnt > 128 {
			b.Skip("Skipping test due to short mode flag")
		}
	}
	prime := cyclic.NewInt(0)
	prime.SetString(PRIME, 16)

	rng := cyclic.NewRandom(cyclic.NewInt(0), cyclic.NewInt(1000))
	grp := cyclic.NewGroup(prime, cyclic.NewInt(5), cyclic.NewInt(4),
		rng)

	rounds := GenerateRounds(nodeCount, batchSize, &grp, b)

	// Rewrite permutation pattern
	for i := 0; i < nodeCount; i++ {
		for j := uint64(0); j < batchSize; j++ {
			// Shift by 1
			newj := (j + 1) % batchSize
			rounds[i].Permutations[j] = newj
		}
	}

	MultiNodePrecomp(nodeCount, batchSize, &grp, rounds, b)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tmpRounds := CopyRounds(nodeCount, rounds)
		inputMsgs,outputMsgs := GenerateIOMessages(nodeCount, batchSize, tmpRounds)

		MultiNodeRealtime(nodeCount, batchSize, &grp, tmpRounds, inputMsgs,
			outputMsgs, b)
	}
}

func BenchmarkRealtime_1_1(b *testing.B) { Realtime(1, 1, b) }
func BenchmarkRealtime_1_2(b *testing.B) { Realtime(1, 2, b) }
func BenchmarkRealtime_1_4(b *testing.B) { Realtime(1, 4, b) }
func BenchmarkRealtime_1_8(b *testing.B) { Realtime(1, 8, b) }
func BenchmarkRealtime_1_16(b *testing.B) { Realtime(1, 16, b) }
func BenchmarkRealtime_1_32(b *testing.B) { Realtime(1, 32, b) }
func BenchmarkRealtime_1_64(b *testing.B) { Realtime(1, 64, b) }
func BenchmarkRealtime_1_128(b *testing.B) { Realtime(1, 128, b) }
func BenchmarkRealtime_1_256(b *testing.B) { Realtime(1, 256, b) }
func BenchmarkRealtime_1_512(b *testing.B) { Realtime(1, 512, b) }
func BenchmarkRealtime_1_1024(b *testing.B) { Realtime(1, 1024, b) }

func BenchmarkRealtime_3_1(b *testing.B) { Realtime(3, 1, b) }
func BenchmarkRealtime_3_2(b *testing.B) { Realtime(3, 2, b) }
func BenchmarkRealtime_3_4(b *testing.B) { Realtime(3, 4, b) }
func BenchmarkRealtime_3_8(b *testing.B) { Realtime(3, 8, b) }
func BenchmarkRealtime_3_16(b *testing.B) { Realtime(3, 16, b) }
func BenchmarkRealtime_3_32(b *testing.B) { Realtime(3, 32, b) }
func BenchmarkRealtime_3_64(b *testing.B) { Realtime(3, 64, b) }
func BenchmarkRealtime_3_128(b *testing.B) { Realtime(3, 128, b) }
func BenchmarkRealtime_3_256(b *testing.B) { Realtime(3, 256, b) }
func BenchmarkRealtime_3_512(b *testing.B) { Realtime(3, 512, b) }
func BenchmarkRealtime_3_1024(b *testing.B) { Realtime(3, 1024, b) }

func BenchmarkRealtime_5_1(b *testing.B) { Realtime(5, 1, b) }
func BenchmarkRealtime_5_2(b *testing.B) { Realtime(5, 2, b) }
func BenchmarkRealtime_5_4(b *testing.B) { Realtime(5, 4, b) }
func BenchmarkRealtime_5_8(b *testing.B) { Realtime(5, 8, b) }
func BenchmarkRealtime_5_16(b *testing.B) { Realtime(5, 16, b) }
func BenchmarkRealtime_5_32(b *testing.B) { Realtime(5, 32, b) }
func BenchmarkRealtime_5_64(b *testing.B) { Realtime(5, 64, b) }
func BenchmarkRealtime_5_128(b *testing.B) { Realtime(5, 128, b) }
func BenchmarkRealtime_5_256(b *testing.B) { Realtime(5, 256, b) }
func BenchmarkRealtime_5_512(b *testing.B) { Realtime(5, 512, b) }
func BenchmarkRealtime_5_1024(b *testing.B) { Realtime(5, 1024, b) }

func BenchmarkRealtime_10_1(b *testing.B) { Realtime(10, 1, b) }
func BenchmarkRealtime_10_2(b *testing.B) { Realtime(10, 2, b) }
func BenchmarkRealtime_10_4(b *testing.B) { Realtime(10, 4, b) }
func BenchmarkRealtime_10_8(b *testing.B) { Realtime(10, 8, b) }
func BenchmarkRealtime_10_16(b *testing.B) { Realtime(10, 16, b) }
func BenchmarkRealtime_10_32(b *testing.B) { Realtime(10, 32, b) }
func BenchmarkRealtime_10_64(b *testing.B) { Realtime(10, 64, b) }
func BenchmarkRealtime_10_128(b *testing.B) { Realtime(10, 128, b) }
func BenchmarkRealtime_10_256(b *testing.B) { Realtime(10, 256, b) }
func BenchmarkRealtime_10_512(b *testing.B) { Realtime(10, 512, b) }
func BenchmarkRealtime_10_1024(b *testing.B) { Realtime(10, 1024, b) }
