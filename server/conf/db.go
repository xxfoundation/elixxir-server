////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package conf

import "github.com/pkg/errors"

type DB struct {
	DBName      string
	DBUserName  string
	DBPassword  string
	DBAddresses []string
	enable      bool
}

// SetDB returns an interface to a DB if all inputs are valid,
// otherwise it returns an error specifying which input was invalid.
func (db *DB) SetDB(dbName, userName, password string, addresses []string) error {

	// Check if SetDB is enabled
	if !db.enable {
		return errors.Errorf("SetDB failed due to improper init.")
	}

	// Check if input fields are valid
	if !isDBNameValid(dbName) {
		return errors.Errorf("SetDB failed with DBName %s", dbName)
	}
	if !isUserNameValid(userName) {
		return errors.Errorf("SetDB failed with DBUserName %s", userName)
	}
	if !isPasswordValid(password) {
		return errors.Errorf("SetDB failed with DBPassword %s", password)
	}

	if addresses == nil {
		return errors.Errorf("SetDB failed with DBAddresses nil")
	}
	for _, address := range addresses {
		if !isAddressValid(address) {
			return errors.Errorf("SetDB failed with DBAddresses %s", address)
		}
	}

	// Set the values
	db.DBName = dbName
	db.DBUserName = userName
	db.DBPassword = password
	db.DBAddresses = addresses

	// Disable updating values
	db.enable = false

	return nil
}

// isDBNameValid returns true for any string
// TODO: Function should return true for any string
// which is a valid database name based on db impl.
func isDBNameValid(dbName string) bool {
	return true
}

// isUserNameValid returns true for any string
// TODO: Function should return true for any
// alphanumeric which doesn't begin with a number
func isUserNameValid(userName string) bool {
	return true
}

// isPasswordValid returns true for any string
// TODO: Change DBPassword to be a secure memguard type
// and modify this function to handle that accordingly
func isPasswordValid(password string) bool {
	return true
}

// isAddressValid returns true for any string
// TODO: Function should check if format matches
// <ip_address>:<port> or some eq. representation.
func isAddressValid(address string) bool {
	return true
}
