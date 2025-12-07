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
)

// SaveNdfToCache saves NDF data to disk cache atomically using temp file + rename pattern
func SaveNdfToCache(ndfData []byte, cachePath string) error {
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

	// Write NDF data to temp file with permissions 0644
	err = os.WriteFile(tmpPath, ndfData, 0644)
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

// LoadNdfFromCache loads NDF data from disk cache
func LoadNdfFromCache(cachePath string) ([]byte, error) {
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

	jww.INFO.Printf("Successfully loaded NDF from cache: %s (%d bytes)", cachePath, len(data))
	return data, nil
}
