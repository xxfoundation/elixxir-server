package server

import (
	"fmt"
	"github.com/spf13/viper"
)

// Run is the main loop for the cMix server
func Run() {
	fmt.Println("Hello, World!")
}

// StartServer reads configuration options and starts the cMix server
func StartServer() {
	viper.Debug()
	Run()
}
