////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

// Handles command-line signing functionality

package cmd

import (
	"crypto/tls"
	"fmt"
	"github.com/InfiniteLoopSpace/go_S-MIME/smime"
	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func init() {
	rootCmd.AddCommand(signCmd)
}

var signCmd = &cobra.Command{
	Use:   "sign",
	Args:  cobra.MinimumNArgs(0),
	Short: "Sign the provided file",
	Long: `Use the XX Node private key to sign files. This produces a
signature file that can be verified with openssl, e.g.:

openssl smime -verify -in [filename].signed -signer [nodecertificate] \
    -CAfile keys/cmix.rip.crt
`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			fmt.Printf("No filenames provided, signing " +
				"default-statement.txt.signed\n")
			out, _ := os.Create("default-statement.txt")
			out.Write([]byte("I am applying to join the BetaNet" +
				" Rollover Program: " + time.Now().String() +
				"\n"))
			out.Close()
			args = []string{"default-statement.txt"}
		}

		fileData, err := ioutil.ReadFile(getKeyAndCertLocationsFile())
		if err != nil {
			jww.FATAL.Panicf("Unable to read key location %s",
				err.Error())
		}
		keyPairPaths := strings.Split(string(fileData), ";")

		keyPair, err := tls.LoadX509KeyPair(string(keyPairPaths[1]),
			string(keyPairPaths[0]))
		if err != nil {
			jww.FATAL.Panicf("Could not load keypair: %+v", err)
		}

		signer, err := smime.New(keyPair)
		if err != nil {
			jww.FATAL.Panicf("Could not load smime: %+v", err)
		}

		fmt.Printf("Please copy & paste the following:\n\n")
		for _, f := range args {
			filedata, err := ioutil.ReadFile(f)
			if err != nil {
				jww.FATAL.Panicf("Could not sign file %s: %s",
					f, err.Error())
			}
			filedata = []byte("Content-Type: text/plain\nContent-Transfer-Encoding: base64\nContent-Disposition: inline\n\n" +
				string(filedata))
			sig, err := signer.Sign(filedata)
			if err != nil {
				jww.FATAL.Panicf("Can't sign file %s: %+v",
					f, err)
			}

			fname := f + ".signed"
			sigFile, err := os.Create(fname)
			if err != nil {
				jww.FATAL.Panicf("Can't create sigfile %s: %s",
					f, err.Error())
			}

			sigFile.Write(sig)
			sigFile.Close()

			fmt.Printf("=====BEGIN=====\n%s\n=====END=====\n", sig)
		}

	},
}

func getKeyAndCertLocationsFile() string {
	filename, err := os.Executable()
	if err != nil {
		jww.ERROR.Printf("Unable to write private key path, "+
			"could not get binary path: %s", err.Error())
	}

	return filename + "-privatekeyandcertpath.txt"
}

// RecordPrivateKeyAndCertPaths determines the location of the binary running this
// function using the os library and stores keyPath into
// "binaryname-privatekeypath.txt"
func RecordPrivateKeyAndCertPaths(keyPath, certPath string) {
	keyPathFile, err := os.Create(getKeyAndCertLocationsFile())
	if err != nil {
		jww.ERROR.Printf("Unable to write private key path, "+
			"could open file for writing: %s", err.Error())
	}

	absKeyPath, err := filepath.Abs(keyPath)
	if err != nil {
		jww.ERROR.Printf("Unable to write private key path, "+
			"could get absolute path: %s", err.Error())
	}
	absCertPath, err := filepath.Abs(certPath)
	if err != nil {
		jww.ERROR.Printf("Unable to write private key path, "+
			"could get absolute path: %s", err.Error())
	}

	keyPathFile.Write([]byte(absKeyPath + ";" + absCertPath))
}
