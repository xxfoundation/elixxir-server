package cmd

import (
  "fmt"

  "github.com/spf13/cobra"
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
    fmt.Println("Hugo Static Site Generator v0.9 -- HEAD")
  },
}
