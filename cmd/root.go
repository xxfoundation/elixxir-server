////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Package cmd initializes the CLI and config parsers as well as the logger.
package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/primitives/utils"
	"os"
	"runtime"
	// net/http must be imported before net/http/pprof for the pprof import
	// to automatically initialize its http handlers
	"net/http"
	_ "net/http/pprof"
)

var cfgFile string
var logLevel uint // 0 = info, 1 = debug, >1 = trace
var validConfig bool
var keepBuffers bool
var logPath = "cmix-server.log"
var maxProcsOverride int
var disableStreaming bool
var useGPU bool
var registrationCode string

// If true, runs pprof http server
var profile bool

// rootCmd represents the base command when called without any sub-commands
var rootCmd = &cobra.Command{
	Use:   "server",
	Short: "Runs a server node for cMix anonymous communication platform",
	Long: `The server provides a full cMix node for distributed anonymous
communications.`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
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
					"0.0.0.0:8087", nil))
			}()
		}

		err := StartServer(viper.GetViper())
		if err != nil {
			jww.FATAL.Panicf("Failed to start server: %+v", err)
		}

		// Prevent node from exiting
		select {}
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
	jww.INFO.Printf("Node exiting without error...")
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
		"Required.  config file (default is $HOME/.elixxir/server.yaml)")
	err := rootCmd.MarkFlagRequired("config")
	handleBindingError(err, "config")

	rootCmd.Flags().UintVarP(&logLevel, "logLevel", "l", 1,
		"Level of debugging to display. 0 = info, 1 = debug, >1 = trace")
	err = viper.BindPFlag("logLevel", rootCmd.Flags().Lookup("logLevel"))
	handleBindingError(err, "logLevel")

	rootCmd.Flags().BoolVar(&profile, "profile", false,
		"Runs a pprof server at 0.0.0.0:8087 for profiling")
	err = rootCmd.Flags().MarkHidden("profile")
	handleBindingError(err, "profile")
	err = viper.BindPFlag("profile", rootCmd.Flags().Lookup("profile"))
	handleBindingError(err, "profile")

	rootCmd.Flags().StringVarP(&registrationCode, "registrationCode", "", "",
		"Required.  Registration code to give to permissioning")
	err = rootCmd.MarkFlagRequired("registrationCode")
	handleBindingError(err, "registrationCode")
	err = viper.BindPFlag("registrationCode", rootCmd.Flags().Lookup("registrationCode"))
	handleBindingError(err, "registrationCode")

	rootCmd.Flags().BoolVarP(&keepBuffers, "keepBuffers", "k", false,
		"maintains all old round information forever, will eventually "+
			"run out of memory")
	err = rootCmd.Flags().MarkHidden("keepBuffers")
	handleBindingError(err, "keepBuffers")
	err = viper.BindPFlag("keepBuffers", rootCmd.Flags().Lookup("keepBuffers"))
	handleBindingError(err, "keepBuffers")

	rootCmd.Flags().IntVar(&maxProcsOverride, "MaxProcsOverride", runtime.NumCPU(),
		"Overrides the maximum number of processes go will use. Must "+
			"be equal to or less than the number of logical cores on the device. "+
			"Defaults at the number of logical cores on the device")
	err = rootCmd.Flags().MarkHidden("MaxProcsOverride")
	handleBindingError(err, "MaxProcsOverride")

	rootCmd.Flags().BoolVarP(&disableStreaming, "disableStreaming", "",
		false, "Disables streaming comms.")
	rootCmd.Flags().BoolVarP(&useGPU, "useGPU", "", false,
		"Toggle on GPU")

	err = viper.BindPFlag("useGPU", rootCmd.Flags().Lookup("useGPU"))
	handleBindingError(err, "useGPU")

}

func handleBindingError(err error, flag string) {
	if err != nil {
		jww.FATAL.Panicf("Error on binding flag \"%s\":%+v", flag, err)
	}
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	//Use default config location if none is passed
	if cfgFile == "" {
		var err error
		cfgFile, err = utils.SearchDefaultLocations("server.yaml","xxnetwork")
		// Find home directory.
		if err!=nil{
			jww.FATAL.Panicf("No config provided and non found at default paths")
		}

	}

	f, err := os.Open(cfgFile)

	_, err = f.Stat()

	validConfig = true

	if err != nil {
		jww.ERROR.Printf("Invalid config file (%s): %s", cfgFile,
			err.Error())
		validConfig = false
	}

	err = f.Close()

	if err != nil {
		jww.ERROR.Printf("Could not close config file: %+v", err)
	}

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
	vipLogLevel := viper.GetUint("logLevel")

	// Check the level of logs to display
	if vipLogLevel > 1 {
		// Set the GRPC log level
		err := os.Setenv("GRPC_GO_LOG_SEVERITY_LEVEL", "info")
		if err != nil {
			jww.ERROR.Printf("Could not set GRPC_GO_LOG_SEVERITY_LEVEL: %+v", err)
		}

		err = os.Setenv("GRPC_GO_LOG_VERBOSITY_LEVEL", "99")
		if err != nil {
			jww.ERROR.Printf("Could not set GRPC_GO_LOG_VERBOSITY_LEVEL: %+v", err)
		}

		// Turn on trace logs
		jww.SetLogThreshold(jww.LevelTrace)
		jww.SetStdoutThreshold(jww.LevelTrace)
		mixmessages.TraceMode()
	} else if vipLogLevel == 1 {
		// Turn on debugging logs
		jww.SetLogThreshold(jww.LevelDebug)
		jww.SetStdoutThreshold(jww.LevelDebug)
		mixmessages.DebugMode()
	} else {
		// Turn on info logs
		jww.SetLogThreshold(jww.LevelInfo)
		jww.SetStdoutThreshold(jww.LevelInfo)
	}

	if viper.Get("node.paths.log") != nil {
		// Create log file, overwrites if existing
		logPath = viper.GetString("node.paths.log")
	} else {
		fmt.Printf("Invalid or missing log path %s, "+
			"default path used.\n", logPath)
	}

	fullLogPath, _ := utils.ExpandPath(logPath)
	logFile, err := os.OpenFile(fullLogPath,
		os.O_CREATE|os.O_WRONLY|os.O_APPEND,
		0644)
	if err != nil {
		fmt.Printf("Could not open log file %s!\n", logPath)
	} else {
		jww.SetLogOutput(logFile)
	}
}
