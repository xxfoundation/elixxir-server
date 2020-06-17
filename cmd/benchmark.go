///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"
	//	"gitlab.com/elixxir/server/benchmark"
	"time"
)

var benchBatchSize uint64
var nodeCount int
var iterations int
var debug bool

func init() {
	// NOTE: The point of init() is to be declarative.
	// There is one init in each sub command. Do not put variable
	// declarations here, and ensure all the Flags are of the *P variety,
	// unless there's a very good reason not to have them as local
	// params to sub command.

	benchmarkCmd.Flags().Uint64VarP(&benchBatchSize, "batch", "b", 1,
		"Batch size to use for node server rounds")
	benchmarkCmd.Flags().IntVarP(&nodeCount, "numnodes", "n", 1,
		"Number of nodes for the benchmark")
	benchmarkCmd.Flags().IntVarP(&iterations, "iterations", "i", 100,
		"Number of times to iterate the benchmark")
	benchmarkCmd.Flags().BoolVarP(&debug, "debug", "", false,
		"Show debug and warning info (default is to only show errors "+
			"and above)")

	rootCmd.AddCommand(benchmarkCmd)

}

var benchmarkCmd = &cobra.Command{
	Use:   "benchmark",
	Short: "Server benchmarking tests",
	Long:  "Run internal benchmark funcs by specifying node & batch sizes",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Running benchmarks for %d nodes with %d batch "+
			"size and %d iterations...\n", nodeCount,
			benchBatchSize, iterations)

		if debug {
			jww.SetLogThreshold(jww.LevelDebug)
		} else {
			jww.SetLogThreshold(jww.LevelError)
		}

		start := time.Now()
		//benchmark.PrecompIterations(nodeCount, benchBatchSize,
		// iterations)
		precompDelta := ((float64(time.Since(start)) / 1000000000) /
			float64(iterations))
		fmt.Printf("Precomp took an average of %f s\n", precompDelta)

		start = time.Now()
		//benchmark.RealtimeIterations(nodeCount, benchBatchSize,
		// iterations)
		realtimeDelta := ((float64(time.Since(start)) / 1000000000) /
			float64(iterations))
		fmt.Printf("Realtime took an average of %f s\n", realtimeDelta)
	},
}
