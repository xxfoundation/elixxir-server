////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Package cmd initializes the CLI and config parsers as well as the logger.
package cmd

import (
	"os"

	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"
	"gitlab.com/privategrity/crypto/forward"
	"math"
)

var cfgFile string
var verbose bool
var noRatchet bool
var serverIdx int
var batchSize uint64
var nodeID uint64

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "server",
	Short: "Runs a server node for cMix anonymous communication platform",
	Long: `The server provides a full cMix node for distributed anonymous
communications.`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if noRatchet {
			forward.SetRatchetStatus(false)
		}
		StartServer(serverIdx, uint64(viper.GetInt("batchsize")))
	},
}

// Execute adds all child commands to the root command and sets flags
// appropriately.  This is called by main.main(). It only needs to
// happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		jww.ERROR.Println(err)
		os.Exit(1)
	}
}

// init is the initialization function for Cobra which defines commands
// and flags.
func init() {
	cobra.OnInitialize(initConfig, initLog)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.
	rootCmd.Flags().StringVar(&cfgFile, "config", "",
		"config file (default is $HOME/.privategrity/server.yaml)")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false,
		"Verbose mode for debugging")
	rootCmd.Flags().BoolVar(&noRatchet, "noratchet", false,
		"Avoid ratcheting the keys for forward secrecy")
	rootCmd.Flags().IntVarP(&serverIdx, "index", "i", 0,
		"Config index to use for local server")
	rootCmd.Flags().Uint64VarP(&batchSize, "batch", "b", 1,
		"Batch size to use for node server rounds")
	rootCmd.Flags().Uint64VarP(&nodeID, "nodeID", "n",
		math.MaxUint64, "Unique identifier for this node")
	viper.BindPFlag("batchSize", rootCmd.Flags().Lookup("batch"))
	viper.BindPFlag("nodeID", rootCmd.Flags().Lookup("nodeID"))
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			jww.ERROR.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name "server" (without extension).
		viper.AddConfigPath(home + "/.privategrity")
		viper.SetConfigName("server")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err != nil {
		jww.ERROR.Printf("Unable to read config file: %s", err)
	}

}

// initLog initializes logging thresholds and the log path.
func initLog() {
	if viper.Get("logPath") != nil {
		// If verbose flag set then log more info for debugging
		if verbose || viper.GetBool("verbose") {
			jww.SetLogThreshold(jww.LevelDebug)
			jww.SetStdoutThreshold(jww.LevelDebug)
		} else {
			jww.SetLogThreshold(jww.LevelInfo)
			jww.SetStdoutThreshold(jww.LevelInfo)
		}
		// Create log file, overwrites if existing
		logPath := viper.GetString("logPath")
		logFile, err := os.Create(logPath)
		if err != nil {
			jww.WARN.Println("Invalid or missing log path, default path used.")
		} else {
			jww.SetLogOutput(logFile)
		}
	}
}
