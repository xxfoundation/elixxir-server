///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

// Handles command-line signing functionality

package cmd

import (
	"crypto"
	"crypto/rand"
	"crypto/sha256"
	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"io/ioutil"
	"os"
	"path/filepath"
	// "encoding/base64"
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

openssl x509 -pubkey -noout -in [node certfile] > pubkey.pem && \
    openssl dgst -sha256 -sigopt rsa_padding_mode:pss -verify pubkey.pem \
    -signature [SignedFile].sig.sha256 [SignedFile]
`,
	Run: func(cmd *cobra.Command, args []string) {
		privateKey := ReadPrivateKey()
		for _, f := range args {
			filedata, err := ioutil.ReadFile(f)
			if err != nil {
				jww.FATAL.Panicf("Could not sign file %s: %s",
					f, err.Error())
			}
			sig := signData(filedata, privateKey)

			sigFile, err := os.Create(f + ".sig.sha256")
			if err != nil {
				jww.FATAL.Panicf("Can't create sigfile %s: %s",
					f, err.Error())
			}
			sigFile.Write(sig)
		}

	},
}

func signData(filedata []byte, privateKey *rsa.PrivateKey) []byte {
	h := sha256.New()
	h.Write(filedata)
	data := h.Sum(nil)
	validSignature, err := rsa.Sign(rand.Reader, privateKey, crypto.SHA256,
		data, nil)
	if err != nil {
		jww.FATAL.Panicf("Failed to sign data: %+v", err)
	}
	return validSignature
}

func getPrivateKeyLocationFile() string {
	filename, err := os.Executable()
	if err != nil {
		jww.ERROR.Printf("Unable to write private key path, "+
			"could not get binary path: %s", err.Error())
	}

	return filename + "-privatekeypath.txt"
}

// ReadPrivateKey path reads and returns a key that can be used for signing
func ReadPrivateKey() *rsa.PrivateKey {
	privateKeyPath, err := ioutil.ReadFile(getPrivateKeyLocationFile())
	if err != nil {
		jww.ERROR.Panicf("Unable to read key location %s", err.Error())
	}
	keyPEMBytes, err := ioutil.ReadFile(string(privateKeyPath))
	if err != nil {
		jww.ERROR.Panicf("Unable to read private key %s", err.Error())
	}

	privateKey, err := rsa.LoadPrivateKeyFromPem(keyPEMBytes)
	if err != nil {
		jww.ERROR.Panicf("Unable to decode private key %s", err.Error())
	}
	return privateKey
}

// RecordPrivateKeyPath determines the location of the binary running this
// function using the os library and stores keyPath into
// "binaryname-privatekeypath.txt"
func RecordPrivateKeyPath(keyPath string) {
	keyPathFile, err := os.Create(getPrivateKeyLocationFile())
	if err != nil {
		jww.ERROR.Printf("Unable to write private key path, "+
			"could open file for writing: %s", err.Error())
	}

	absKeyPath, err := filepath.Abs(keyPath)
	if err != nil {
		jww.ERROR.Printf("Unable to write private key path, "+
			"could get absolute path: %s", err.Error())
	}
	keyPathFile.Write([]byte(absKeyPath))
}
