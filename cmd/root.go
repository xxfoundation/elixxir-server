////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Package cmd initializes the CLI and config parsers as well as the logger.
package cmd

import (
	//"gitlab.com/elixxir/server/globals"
	"os"
	"time"

	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"
	"math"

	// net/http must be imported before net/http/pprof for the pprof import
	// to automatically initialize its http handlers
	"net/http"
	_ "net/http/pprof"
)

var cfgFile string
var verbose bool
var serverIdx int
var batchSize uint64
var nodeID uint64
var validConfig bool
var showVer bool

// If true, runs pprof http server
var profile bool

var roundBufferTimeout time.Duration

// rootCmd represents the base command when called without any sub-commands
var rootCmd = &cobra.Command{
	Use:   "server",
	Short: "Runs a server node for cMix anonymous communication platform",
	Long: `The server provides a full cMix node for distributed anonymous
communications.`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if showVer {
			printVersion()
			return
		}
		if !validConfig {
			jww.FATAL.Panic("Invalid Config File")
		}
		if profile {
			go func() {
				// Do not expose this port over the
				// network by serving on ":8087" or
				// "0.0.0.0:8087". If you wish to profile
				// production servers, do it by SSHing
				// into the server and using go tool
				// pprof. This provides simple access
				// control for the profiling
				jww.FATAL.Println(http.ListenAndServe(
					"localhost:8087", nil))
			}()
		}

		StartServer(viper.GetViper())
	},
}

// Execute adds all child commands to the root command and sets flags
// appropriately.  This is called by main.main(). It only needs to
// happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		jww.ERROR.Printf("Node Exiting with error: %s", err.Error())
		os.Exit(1)
	}

}

// init is the initialization function for Cobra which defines commands
// and flags.
func init() {
	// NOTE: The point of init() is to be declarative.  There
	// is one init in each sub command. Do not put variable
	// declarations here, and ensure all the Flags are of the *P
	// variety, unless there's a very good reason not to have them
	// as local params to sub command."
	cobra.OnInitialize(initConfig, initLog)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.
	rootCmd.Flags().StringVarP(&cfgFile, "config", "", "",
		"config file (default is $HOME/.elixxir/server.yaml)")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false,
		"Verbose mode for debugging")
	rootCmd.Flags().IntVarP(&serverIdx, "index", "i", 0,
		"Config index to use for local server")
	rootCmd.Flags().Uint64VarP(&batchSize, "batch", "b", 1,
		"Batch size to use for node server rounds")
	rootCmd.Flags().Uint64VarP(&nodeID, "nodeID", "n",
		math.MaxUint64, "Unique identifier for this node")
	rootCmd.Flags().BoolVarP(&showVer, "version", "V", false,
		"Show the server version information.")
	rootCmd.Flags().BoolVar(&profile, "profile", false,
		"Runs a pprof server at localhost:8087 for profiling")
	rootCmd.Flags().DurationVar(&roundBufferTimeout, "roundBufferTimeout",
		time.Second, "Determines the amount of time the  GetRoundBufferInfo"+
			" RPC will wait before returning an error")

	viper.BindPFlag("batchSize", rootCmd.Flags().Lookup("batch"))
	viper.BindPFlag("nodeID", rootCmd.Flags().Lookup("nodeID"))
	viper.BindPFlag("profile", rootCmd.Flags().Lookup("profile"))
	viper.BindPFlag("index", rootCmd.Flags().Lookup("index"))
	viper.BindPFlag("roundBufferTimeout", rootCmd.Flags().Lookup("roundBufferTimeout"))
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	//Use default config location if none is passed
	if cfgFile == "" {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			jww.ERROR.Println(err)
			os.Exit(1)
		}

		cfgFile = home + "/.elixxir/server.yaml"

	}

	f, err := os.Open(cfgFile)

	_, err = f.Stat()

	validConfig = true

	if err != nil {
		jww.ERROR.Printf("Invalid config file (%s): %s", cfgFile,
			err.Error())
		validConfig = false
	}

	f.Close()

	viper.SetConfigFile(cfgFile)

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err = viper.ReadInConfig(); err != nil {
		jww.ERROR.Printf("Unable to read config file (%s): %s", cfgFile,
			err.Error())
		validConfig = false
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
			jww.WARN.Println("Invalid or missing log path, " +
				"default path used.")
		} else {
			jww.SetLogOutput(logFile)
		}
	}
}
