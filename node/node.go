package node

import (
	"fmt"
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"
)

// Run is the main loop for the cMix server
func Run() {
	fmt.Println("Hello, World!")
}

// StartServer reads configuration options and starts the cMix server
// Create directory ".privategrity"
func StartServer() {
	viper.Debug()
	jww.INFO.Printf("Log Filename: %v\n", viper.GetString("logPath"))
	jww.INFO.Printf("Config Filename: %v\n", viper.ConfigFileUsed())
	jww.ERROR.Println("Logger works!")
	jww.INFO.Println("Verbose works!")
	Run()
}
