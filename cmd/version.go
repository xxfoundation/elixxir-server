////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
)

//go:generate go run gen.go
// The above generates: GITVERSION, GLIDEDEPS, and SEMVER

func init() {
	rootCmd.AddCommand(versionCmd)
}

func printVersion() {
	fmt.Printf(getVersionInfo())
}

func getVersionInfo() string {
	version := fmt.Sprintf("Elixxir Server v%s -- %s\n\n", SEMVER,
		GITVERSION)
	version = fmt.Sprintf("%sDependencies:\n\n%s\n", version, GLIDEDEPS)
	return version
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of Elixxir Server",
	Long: `Print the version number of Elixxir Server. This also prints
the glide cache versions of all of its dependencies.`,
	Run: func(cmd *cobra.Command, args []string) {
		printVersion()
	},
}
