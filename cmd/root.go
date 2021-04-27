///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

// Package cmd initializes the CLI and config parsers as well as the logger.
package cmd

import (
	"flag"
	"fmt"
	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/xx_network/primitives/utils"
	"os"
	"runtime"
	"strings"
	// net/http must be imported before net/http/pprof for the pprof import
	// to automatically initialize its http handlers
	"net/http"
	_ "net/http/pprof"
	"os/signal"
	"time"
)

var cfgFile string
var logLevel uint // 0 = info, 1 = debug, >1 = trace
var validConfig bool
var keepBuffers bool
var logPath = "cmix-server.log"
var maxProcsOverride int
var disableStreaming bool
var useGPU bool
var BatchSizeGPUTest int

// If true, runs pprof http server
var profile bool

// rootCmd represents the base command when called without any sub-commands
var rootCmd = &cobra.Command{
	Use:   "server",
	Short: "Runs a server node for cMix anonymous communication platform",
	Long:  `The server provides a full cMix node for distributed anonymous communications.`,
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		initConfig()
		initLog()
		if !validConfig {
			jww.FATAL.Panicf("Invalid Config File: %s", cfgFile)
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

		jww.INFO.Printf("Starting xx network node (server) v%s", SEMVER)

		instance, err := StartServer(viper.GetViper())
		// Retry to start the instance on certain errors
		for {
			if err == nil {
				break
			}
			errMsg := err.Error()
			transport := strings.Contains(errMsg, "transport is closing")
			cde := strings.Contains(errMsg, "DeadlineExceeded")
			ndf := strings.Contains(errMsg, "ndf")
			iot := strings.Contains(errMsg, "i/o timeout")
			if (ndf && (cde || transport)) || iot {
				if instance != nil && instance.GetNetwork() != nil {
					instance.GetNetwork().Shutdown()
				}
				jww.ERROR.Print("Cannot start, permissioning " +
					"is unavailable, retrying in 10s...")
				time.Sleep(10 * time.Second)
				instance, err = StartServer(viper.GetViper())
				continue
			}
			jww.FATAL.Panicf("Failed to start server: %+v",
				err)
		}
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		signal.Notify(c, os.Kill)
		// Prevent node from exiting
		select {
		case <-c:
			jww.ERROR.Printf("Exiting due to an exit signal...")
			instance.Shutdown()
		}
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

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.
	rootCmd.Flags().StringVarP(&cfgFile, "config", "c", "",
		"Path to load the Node configuration file from. If not set, this "+
			"file must be named gateway.yaml and must be located in "+
			"~/.xxnetwork/, /opt/xxnetwork, or /etc/xxnetwork.")

	rootCmd.Flags().UintVarP(&logLevel, "logLevel", "l", 0,
		"Level of debugging to print (0 = info, 1 = debug, >1 = trace).")
	err := viper.BindPFlag("logLevel", rootCmd.Flags().Lookup("logLevel"))
	handleBindingError(err, "logLevel")

	rootCmd.Flags().BoolVar(&profile, "profile", false,
		"Runs a pprof server at 0.0.0.0:8087 for profiling.")
	err = rootCmd.Flags().MarkHidden("profile")
	handleBindingError(err, "profile")
	err = viper.BindPFlag("profile", rootCmd.Flags().Lookup("profile"))
	handleBindingError(err, "profile")

	rootCmd.Flags().String("registrationCode", "",
		"Registration code used for first time registration. This is a unique "+
			"code provided by xx network. (Required)")
	err = viper.BindPFlag("registrationCode", rootCmd.Flags().Lookup("registrationCode"))
	handleBindingError(err, "registrationCode")

	rootCmd.Flags().BoolVarP(&keepBuffers, "keepBuffers", "k", false,
		"Maintains all of the old round information forever; will eventually "+
			"run out of memory.")
	err = rootCmd.Flags().MarkHidden("keepBuffers")
	handleBindingError(err, "keepBuffers")
	err = viper.BindPFlag("keepBuffers", rootCmd.Flags().Lookup("keepBuffers"))
	handleBindingError(err, "keepBuffers")

	rootCmd.Flags().IntVar(&maxProcsOverride, "MaxProcsOverride", runtime.NumCPU(),
		"Overrides the maximum number of processes Go will use. Must be equal "+
			"to or less than the number of logical cores on the device. "+
			"Defaults at the number of logical cores on the device.")
	err = rootCmd.Flags().MarkHidden("MaxProcsOverride")
	handleBindingError(err, "MaxProcsOverride")

	rootCmd.Flags().BoolVar(&disableStreaming, "disableStreaming", false,
		"Disables streaming comms.")

	rootCmd.Flags().BoolVar(&useGPU, "useGPU", true, "Toggles use of the GPU.")
	err = viper.BindPFlag("useGPU", rootCmd.Flags().Lookup("useGPU"))
	handleBindingError(err, "useGPU")

	// Gets flag for the batch size used in Test_MultiInstance_N3_B32_GPU
	flag.IntVar(&BatchSizeGPUTest, "batchSize", 0,
		"The batch size used in the multi-instance GPU test.")

	// NOTE: Meant for use by developer team ONLY. The development/maintenance
	// team are NOT responsible for any issues encountered by any users
	// who modify this logic or who run on the network with this option
	rootCmd.Flags().Bool("devMode", false,
		"Run in development/testing mode. Do not use on beta or main nets.")
	err = rootCmd.Flags().MarkHidden("devMode")
	handleBindingError(err, "devMode")
	err = viper.BindPFlag("devMode", rootCmd.Flags().Lookup("devMode"))
	handleBindingError(err, "devMode")

}

func handleBindingError(err error, flag string) {
	if err != nil {
		jww.FATAL.Panicf("Error on binding flag \"%s\":%+v", flag, err)
	}
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	// Use default config location if none is passed
	if cfgFile == "" {
		jww.FATAL.Panicf("No config file provided.")
	}

	f, err := os.Open(cfgFile)
	if err != nil {
		jww.ERROR.Printf("Could not open config file: %+v", err)
		return
	}
	_, err = f.Stat()
	if err != nil {
		jww.ERROR.Printf("Could not stat config file: %+v", err)
		return
	}

	err = f.Close()
	if err != nil {
		jww.ERROR.Printf("Could not close config file: %+v", err)
		return
	}

	viper.SetConfigFile(cfgFile)

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err = viper.ReadInConfig(); err != nil {
		jww.ERROR.Printf("Unable to read config file (%s): %s", cfgFile,
			err.Error())
		return
	}

	validConfig = true
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
