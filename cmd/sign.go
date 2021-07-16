///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

// Handles command-line signing functionality

package cmd

import (
	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"
	// "gitlab.com/xx_network/crypto/signature/rsa"
	"io/ioutil"
	"os"
	"path/filepath"
	// cms "github.com/github/ietf-cms"
	// "crypto/x509"
	// "encoding/base64"
	"strings"
	// "gitlab.com/xx_network/crypto/tls"
	"crypto/tls"
	// "encoding/pem"
	"github.com/InfiniteLoopSpace/go_S-MIME/smime"
)

func init() {
	rootCmd.AddCommand(signCmd)
}

var signCmd = &cobra.Command{
	Use:   "sign",
	Args:  cobra.MinimumNArgs(1),
	Short: "Sign the provided file",
	Long: `Use the XX Node private key to sign files. This produces a
signature file that can be verified with openssl, e.g.:

openssl smime -verify -in [filename].signed -signer [nodecertificate] \
    -CAfile keys/cmix.rip.crt
`,
	Run: func(cmd *cobra.Command, args []string) {
		// Note: API expects native rsa keys pointers
		// privateKey := &readPrivateKey().PrivateKey
		// cer := readCertificate()
		// chain := []*x509.Certificate{cer}

		fileData, err := ioutil.ReadFile(getKeyAndCertLocationsFile())
		if err != nil {
			jww.ERROR.Panicf("Unable to read key location %s",
				err.Error())
		}
		keyPairPaths := strings.Split(string(fileData), ";")
		jww.ERROR.Printf("keyPairPaths: %s\n", keyPairPaths)

		keyPair, err := tls.LoadX509KeyPair(string(keyPairPaths[1]),
			string(keyPairPaths[0]))
		if err != nil {
			jww.FATAL.Panicf("Could not load keypair: %+v", err)
		}

		signer, err := smime.New(keyPair)
		if err != nil {
			jww.FATAL.Panicf("Could not load smime: %+v", err)
		}

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

			sigFile, err := os.Create(f + ".signed")
			if err != nil {
				jww.FATAL.Panicf("Can't create sigfile %s: %s",
					f, err.Error())
			}

			sigFile.Write(sig)
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
