////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

// Handles the high level storage API.
// This layer merges the business logic layer and the database layer

package storage

// Storage API for the storage layer
type Storage struct {
	// Stored database interface
	database
}

// NewStorage Create a new Storage object wrapping a database interface
// Returns a Storage object, close function, and error
func NewStorage(username, password, dbName, address, port string, devMode bool) (*Storage, error) {
	db, err := newDatabase(username, password, dbName, address, port, devMode)
	storage := &Storage{db}
	return storage, err
}
