////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package conf

import "github.com/pkg/errors"

type DB interface {
	GetDBName() string
	GetUserName() string
	GetPassword() string // TODO: maybe this should be a secure string via memguard
	GetAddresses() []string
}

type dbImpl struct {
	dbName    string
	userName  string
	password  string
	addresses []string
}

// NewDB returns an interface to a DB if all inputs are valid,
// otherwise it returns an error specifying which input was invalid.
func NewDB(dbName, userName, password string, addresses []string) (DB, error) {

	// If input fields are not valid return an error.
	if !isDBNameValid(dbName) {
		return nil, errors.Errorf("NewDB failed with dbName %s", dbName)
	}
	if !isUserNameValid(userName) {
		return nil, errors.Errorf("NewDB failed with userName %s", userName)
	}
	if !isPasswordValid(password) {
		return nil, errors.Errorf("NewDB failed with password %s", password)
	}

	if addresses == nil {
		return nil, errors.Errorf("NewDB failed with addresses nil")
	}
	for _, address := range addresses {
		if !isAddressValid(address) {
			return nil, errors.Errorf("NewDB failed with addresses %s", address)
		}
	}

	// Otherwise return the interface to the object with no error
	return dbImpl{
		dbName:    dbName,
		userName:  userName,
		password:  password,
		addresses: addresses,
	}, nil
}

// GetDBName returns the stored database schema name
func (db dbImpl) GetDBName() string {
	return db.dbName
}

// GetAddresses
func (db dbImpl) GetAddresses() []string {
	return db.addresses
}

// GetUserName returns the stored user name for db login
func (db dbImpl) GetUserName() string {
	return db.userName
}

// GetPassword returns the stored password for db login
func (db dbImpl) GetPassword() string {
	return db.password
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
// TODO: Change password to be a secure memguard type
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
