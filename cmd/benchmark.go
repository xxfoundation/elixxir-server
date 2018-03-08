package cmd

import (
	"gitlab.com/privategrity/server/benchmark"
	"github.com/spf13/cobra"
	"time"
	"fmt"
)

var benchBatchSize uint64
var nodeCount int
var iterations int

func init() {
	benchmarkCmd.Flags().Uint64VarP(&benchBatchSize, "batch", "b", 1,
		"Batch size to use for node server rounds")
	benchmarkCmd.Flags().IntVarP(&nodeCount, "numnodes", "n", 1,
		"Number of nodes for the benchmark")
	benchmarkCmd.Flags().IntVarP(&iterations, "iterations", "i", 100,
		"Number of times to iterate the benchmark")

  rootCmd.AddCommand(benchmarkCmd)
}

var benchmarkCmd = &cobra.Command{
  Use:   "benchmark",
  Short: "Server benchmarking tests",
  Long:  `Run internal benchmark funcs by specifying node and batch sizes`,
  Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Running benchmarks for %d nodes with %d batch size and %d" +
			" iterations...\n", nodeCount, benchBatchSize, iterations)

		start := time.Now()
		benchmark.PrecompIterations(nodeCount, benchBatchSize, iterations)
		precompDelta := (float64(time.Since(start))/1000000000)/float64(iterations)
		fmt.Printf("Precomp took an average of %f s\n", precompDelta)

		start = time.Now()
		benchmark.RealtimeIterations(nodeCount, benchBatchSize, iterations)
		realtimeDelta := (float64(time.Since(start))/1000000000)/float64(iterations)
		fmt.Printf("Realtime took an average of %f s\n", realtimeDelta)
  },
}
