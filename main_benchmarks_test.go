////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package main

import (
	"gitlab.com/privategrity/server/benchmark"
	"testing"
)

// GenerateRoundsBenchmarkTests
func BenchmarkGenerateRounds_5_1024(b *testing.B) {
	benchmark.RoundGeneratorBenchmark(5, 1024, b)
}

func BenchmarkPrecomp_1_1(b *testing.B)    { benchmark.Precomp(1, 1, b) }
func BenchmarkPrecomp_1_2(b *testing.B)    { benchmark.Precomp(1, 2, b) }
func BenchmarkPrecomp_1_4(b *testing.B)    { benchmark.Precomp(1, 4, b) }
func BenchmarkPrecomp_1_8(b *testing.B)    { benchmark.Precomp(1, 8, b) }
func BenchmarkPrecomp_1_16(b *testing.B)   { benchmark.Precomp(1, 16, b) }
func BenchmarkPrecomp_1_32(b *testing.B)   { benchmark.Precomp(1, 32, b) }
func BenchmarkPrecomp_1_64(b *testing.B)   { benchmark.Precomp(1, 64, b) }
func BenchmarkPrecomp_1_128(b *testing.B)  { benchmark.Precomp(1, 128, b) }
func BenchmarkPrecomp_1_256(b *testing.B)  { benchmark.Precomp(1, 256, b) }
func BenchmarkPrecomp_1_512(b *testing.B)  { benchmark.Precomp(1, 512, b) }
func BenchmarkPrecomp_1_1024(b *testing.B) { benchmark.Precomp(1, 1024, b) }

func BenchmarkPrecomp_3_1(b *testing.B)    { benchmark.Precomp(3, 1, b) }
func BenchmarkPrecomp_3_2(b *testing.B)    { benchmark.Precomp(3, 2, b) }
func BenchmarkPrecomp_3_4(b *testing.B)    { benchmark.Precomp(3, 4, b) }
func BenchmarkPrecomp_3_8(b *testing.B)    { benchmark.Precomp(3, 8, b) }
func BenchmarkPrecomp_3_16(b *testing.B)   { benchmark.Precomp(3, 16, b) }
func BenchmarkPrecomp_3_32(b *testing.B)   { benchmark.Precomp(3, 32, b) }
func BenchmarkPrecomp_3_64(b *testing.B)   { benchmark.Precomp(3, 64, b) }
func BenchmarkPrecomp_3_128(b *testing.B)  { benchmark.Precomp(3, 128, b) }
func BenchmarkPrecomp_3_256(b *testing.B)  { benchmark.Precomp(3, 256, b) }
func BenchmarkPrecomp_3_512(b *testing.B)  { benchmark.Precomp(3, 512, b) }
func BenchmarkPrecomp_3_1024(b *testing.B) { benchmark.Precomp(3, 1024, b) }

func BenchmarkPrecomp_5_1(b *testing.B)    { benchmark.Precomp(5, 1, b) }
func BenchmarkPrecomp_5_2(b *testing.B)    { benchmark.Precomp(5, 2, b) }
func BenchmarkPrecomp_5_4(b *testing.B)    { benchmark.Precomp(5, 4, b) }
func BenchmarkPrecomp_5_8(b *testing.B)    { benchmark.Precomp(5, 8, b) }
func BenchmarkPrecomp_5_16(b *testing.B)   { benchmark.Precomp(5, 16, b) }
func BenchmarkPrecomp_5_32(b *testing.B)   { benchmark.Precomp(5, 32, b) }
func BenchmarkPrecomp_5_64(b *testing.B)   { benchmark.Precomp(5, 64, b) }
func BenchmarkPrecomp_5_128(b *testing.B)  { benchmark.Precomp(5, 128, b) }
func BenchmarkPrecomp_5_256(b *testing.B)  { benchmark.Precomp(5, 256, b) }
func BenchmarkPrecomp_5_512(b *testing.B)  { benchmark.Precomp(5, 512, b) }
func BenchmarkPrecomp_5_1024(b *testing.B) { benchmark.Precomp(5, 1024, b) }

func BenchmarkPrecomp_10_1(b *testing.B)    { benchmark.Precomp(10, 1, b) }
func BenchmarkPrecomp_10_2(b *testing.B)    { benchmark.Precomp(10, 2, b) }
func BenchmarkPrecomp_10_4(b *testing.B)    { benchmark.Precomp(10, 4, b) }
func BenchmarkPrecomp_10_8(b *testing.B)    { benchmark.Precomp(10, 8, b) }
func BenchmarkPrecomp_10_16(b *testing.B)   { benchmark.Precomp(10, 16, b) }
func BenchmarkPrecomp_10_32(b *testing.B)   { benchmark.Precomp(10, 32, b) }
func BenchmarkPrecomp_10_64(b *testing.B)   { benchmark.Precomp(10, 64, b) }
func BenchmarkPrecomp_10_128(b *testing.B)  { benchmark.Precomp(10, 128, b) }
func BenchmarkPrecomp_10_256(b *testing.B)  { benchmark.Precomp(10, 256, b) }
func BenchmarkPrecomp_10_512(b *testing.B)  { benchmark.Precomp(10, 512, b) }
func BenchmarkPrecomp_10_1024(b *testing.B) { benchmark.Precomp(10, 1024, b) }

func BenchmarkRealtime_1_1(b *testing.B)    { benchmark.Realtime(1, 1, b) }
func BenchmarkRealtime_1_2(b *testing.B)    { benchmark.Realtime(1, 2, b) }
func BenchmarkRealtime_1_4(b *testing.B)    { benchmark.Realtime(1, 4, b) }
func BenchmarkRealtime_1_8(b *testing.B)    { benchmark.Realtime(1, 8, b) }
func BenchmarkRealtime_1_16(b *testing.B)   { benchmark.Realtime(1, 16, b) }
func BenchmarkRealtime_1_32(b *testing.B)   { benchmark.Realtime(1, 32, b) }
func BenchmarkRealtime_1_64(b *testing.B)   { benchmark.Realtime(1, 64, b) }
func BenchmarkRealtime_1_128(b *testing.B)  { benchmark.Realtime(1, 128, b) }
func BenchmarkRealtime_1_256(b *testing.B)  { benchmark.Realtime(1, 256, b) }
func BenchmarkRealtime_1_512(b *testing.B)  { benchmark.Realtime(1, 512, b) }
func BenchmarkRealtime_1_1024(b *testing.B) { benchmark.Realtime(1, 1024, b) }

func BenchmarkRealtime_3_1(b *testing.B)    { benchmark.Realtime(3, 1, b) }
func BenchmarkRealtime_3_2(b *testing.B)    { benchmark.Realtime(3, 2, b) }
func BenchmarkRealtime_3_4(b *testing.B)    { benchmark.Realtime(3, 4, b) }
func BenchmarkRealtime_3_8(b *testing.B)    { benchmark.Realtime(3, 8, b) }
func BenchmarkRealtime_3_16(b *testing.B)   { benchmark.Realtime(3, 16, b) }
func BenchmarkRealtime_3_32(b *testing.B)   { benchmark.Realtime(3, 32, b) }
func BenchmarkRealtime_3_64(b *testing.B)   { benchmark.Realtime(3, 64, b) }
func BenchmarkRealtime_3_128(b *testing.B)  { benchmark.Realtime(3, 128, b) }
func BenchmarkRealtime_3_256(b *testing.B)  { benchmark.Realtime(3, 256, b) }
func BenchmarkRealtime_3_512(b *testing.B)  { benchmark.Realtime(3, 512, b) }
func BenchmarkRealtime_3_1024(b *testing.B) { benchmark.Realtime(3, 1024, b) }

func BenchmarkRealtime_5_1(b *testing.B)    { benchmark.Realtime(5, 1, b) }
func BenchmarkRealtime_5_2(b *testing.B)    { benchmark.Realtime(5, 2, b) }
func BenchmarkRealtime_5_4(b *testing.B)    { benchmark.Realtime(5, 4, b) }
func BenchmarkRealtime_5_8(b *testing.B)    { benchmark.Realtime(5, 8, b) }
func BenchmarkRealtime_5_16(b *testing.B)   { benchmark.Realtime(5, 16, b) }
func BenchmarkRealtime_5_32(b *testing.B)   { benchmark.Realtime(5, 32, b) }
func BenchmarkRealtime_5_64(b *testing.B)   { benchmark.Realtime(5, 64, b) }
func BenchmarkRealtime_5_128(b *testing.B)  { benchmark.Realtime(5, 128, b) }
func BenchmarkRealtime_5_256(b *testing.B)  { benchmark.Realtime(5, 256, b) }
func BenchmarkRealtime_5_512(b *testing.B)  { benchmark.Realtime(5, 512, b) }
func BenchmarkRealtime_5_1024(b *testing.B) { benchmark.Realtime(5, 1024, b) }

func BenchmarkRealtime_10_1(b *testing.B)    { benchmark.Realtime(10, 1, b) }
func BenchmarkRealtime_10_2(b *testing.B)    { benchmark.Realtime(10, 2, b) }
func BenchmarkRealtime_10_4(b *testing.B)    { benchmark.Realtime(10, 4, b) }
func BenchmarkRealtime_10_8(b *testing.B)    { benchmark.Realtime(10, 8, b) }
func BenchmarkRealtime_10_16(b *testing.B)   { benchmark.Realtime(10, 16, b) }
func BenchmarkRealtime_10_32(b *testing.B)   { benchmark.Realtime(10, 32, b) }
func BenchmarkRealtime_10_64(b *testing.B)   { benchmark.Realtime(10, 64, b) }
func BenchmarkRealtime_10_128(b *testing.B)  { benchmark.Realtime(10, 128, b) }
func BenchmarkRealtime_10_256(b *testing.B)  { benchmark.Realtime(10, 256, b) }
func BenchmarkRealtime_10_512(b *testing.B)  { benchmark.Realtime(10, 512, b) }
func BenchmarkRealtime_10_1024(b *testing.B) { benchmark.Realtime(10, 1024, b) }
