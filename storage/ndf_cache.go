////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package storage

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/xx_network/crypto/chacha"
	"gitlab.com/xx_network/crypto/csprng"
)

// SaveNdfToCache saves NDF data to disk cache atomically using temp file + rename pattern
// Also saves the hash in a separate file for cache validation
func SaveNdfToCache(ndfData []byte, cachePath string, key []byte, rng csprng.Source) error {
	// Extract directory from cache path
	dir := filepath.Dir(cachePath)

	// Create directory if it doesn't exist
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		jww.WARN.Printf("Failed to create cache directory: %+v", err)
		return errors.WithMessage(err, "failed to create cache directory")
	}

	// Create temporary file path
	tmpPath := cachePath + ".tmp"

	// Encrypt the NDF data
	encryptedData, err := chacha.Encrypt(key, ndfData, rng)
	if err != nil {
		jww.WARN.Printf("Failed to encrypt NDF data: %+v", err)
		return errors.WithMessage(err, "failed to encrypt NDF data")
	}

	// Write NDF data to temp file with permissions 0644
	err = os.WriteFile(tmpPath, encryptedData, 0644)
	if err != nil {
		jww.WARN.Printf("Failed to write temp file: %+v", err)
		return errors.WithMessage(err, "failed to write temp cache file")
	}

	// Atomically rename temp file to final cache path
	err = os.Rename(tmpPath, cachePath)
	if err != nil {
		// Clean up temp file on failure
		os.Remove(tmpPath)
		jww.WARN.Printf("Failed to rename temp file to cache path: %+v", err)
		return errors.WithMessage(err, "failed to atomically rename cache file")
	}

	jww.INFO.Printf("Successfully cached NDF to %s (%d bytes)", cachePath, len(ndfData))
	return nil
}

// SaveNdfHashToCache saves the NDF hash to a separate cache file
func SaveNdfHashToCache(hash []byte, cachePath string, key []byte, rng csprng.Source) error {
	hashPath := cachePath + ".hash"

	// Encrypt the hash data
	encryptedHash, err := chacha.Encrypt(key, hash, rng)
	if err != nil {
		jww.WARN.Printf("Failed to encrypt NDF hash: %+v", err)
		return errors.WithMessage(err, "failed to encrypt NDF hash")
	}

	// Write hash to file atomically
	tmpPath := hashPath + ".tmp"
	err = os.WriteFile(tmpPath, encryptedHash, 0644)
	if err != nil {
		jww.WARN.Printf("Failed to write hash temp file: %+v", err)
		return errors.WithMessage(err, "failed to write temp hash file")
	}

	err = os.Rename(tmpPath, hashPath)
	if err != nil {
		os.Remove(tmpPath)
		jww.WARN.Printf("Failed to rename hash temp file: %+v", err)
		return errors.WithMessage(err, "failed to atomically rename hash file")
	}

	jww.DEBUG.Printf("Successfully cached NDF hash to %s", hashPath)
	return nil
}

// LoadNdfFromCache loads NDF data from disk cache
func LoadNdfFromCache(cachePath string, key []byte) ([]byte, error) {
	// Check if file exists
	fileInfo, err := os.Stat(cachePath)
	if err != nil {
		if os.IsNotExist(err) {
			jww.DEBUG.Printf("Cache file does not exist: %s", cachePath)
		} else {
			jww.DEBUG.Printf("Cache miss for %s: %+v", cachePath, err)
		}
		return nil, err
	}

	jww.DEBUG.Printf("Cache file found: %s (%d bytes)", cachePath, fileInfo.Size())

	// Read entire file
	data, err := os.ReadFile(cachePath)
	if err != nil {
		jww.DEBUG.Printf("Failed to read cache file: %+v", err)
		return nil, errors.WithMessage(err, "failed to read cache file")
	}

	// Validate non-empty
	if len(data) == 0 {
		jww.DEBUG.Printf("Cache file is empty: %s", cachePath)
		return nil, errors.New("cache file is empty")
	}

	// Decrypt the NDF data
	decryptedData, err := chacha.Decrypt(key, data)
	if err != nil {
		jww.DEBUG.Printf("Failed to decrypt cache file: %+v", err)
		return nil, errors.WithMessage(err, "failed to decrypt cache file")
	}

	jww.INFO.Printf("Successfully loaded NDF from cache: %s (%d bytes)", cachePath, len(decryptedData))
	return decryptedData, nil
}

// LoadNdfHashFromCache loads the NDF hash from the separate cache file
func LoadNdfHashFromCache(cachePath string, key []byte) ([]byte, error) {
	hashPath := cachePath + ".hash"

	// Check if hash file exists
	_, err := os.Stat(hashPath)
	if err != nil {
		if os.IsNotExist(err) {
			jww.DEBUG.Printf("Hash cache file does not exist: %s", hashPath)
		} else {
			jww.DEBUG.Printf("Hash cache miss for %s: %+v", hashPath, err)
		}
		return nil, err
	}

	// Read hash file
	data, err := os.ReadFile(hashPath)
	if err != nil {
		jww.DEBUG.Printf("Failed to read hash cache file: %+v", err)
		return nil, errors.WithMessage(err, "failed to read hash cache file")
	}

	// Decrypt the hash
	decryptedHash, err := chacha.Decrypt(key, data)
	if err != nil {
		jww.DEBUG.Printf("Failed to decrypt hash cache file: %+v", err)
		return nil, errors.WithMessage(err, "failed to decrypt hash cache file")
	}

	jww.DEBUG.Printf("Successfully loaded NDF hash from cache: %s", hashPath)
	return decryptedHash, nil
}
