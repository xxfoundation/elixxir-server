////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package conf

import "testing"

const ValidDBName = "ValidDBName123"
const ValidUserName = "ValidUserName123"
const ValidPassword = "Z8X:6d*n$9A)YQr5"

var ValidAddresses = []string{"127.0.0.1:5000", "127.0.0.1:5001"}

// NewDB should return an error on empty or non-alpha db name
func TestNewDB_ReturnsErrorOnInvalidDBName(t *testing.T) {
	invalidDBNames := []string{"", "#@#$#@"}
	userName := ValidUserName
	password := ValidPassword
	addresses := ValidAddresses

	for _, invalidDBName := range invalidDBNames {

		_, err := NewDB(invalidDBName, userName, password, addresses)

		if err == nil {
			t.Errorf("NewDB did not return an error for dbName %s", invalidDBName)
		}
	}
}

// NewDB should return an error on empty or non-alpha username
func TestNewDB_ReturnsErrorOnInvalidUserName(t *testing.T) {
	invalidUserNames := []string{"", "#@#$#@", "0123"}

	dbName := ValidDBName
	password := ValidPassword
	addresses := ValidAddresses

	for _, invalidUserName := range invalidUserNames {

		_, err := NewDB(dbName, invalidUserName, password, addresses)

		if err == nil {
			t.Errorf("NewDB did not return an error for username %s", invalidUserName)
		}
	}
}

// NewDB should return an error on a long password
func TestNewDB_ReturnsErrorOnInvalidPassword(t *testing.T) {
	invalidPasswords := []string{`
		àbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbc
		àbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbc
		àbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbc
		àbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbc
		àbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbc
		àbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbc
		àbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbc
		àbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbc
		àbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbc
		àbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbc
		àbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbc
		àbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbc
		àbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbc
		àbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbc
		àbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbc
		àbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbc
		àbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbc
		àbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbc
		àbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbc
		àbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbc
		àbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbc
		àbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbc
		àbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbcàbc
	`}

	dbName := ValidDBName
	userName := ValidUserName
	addresses := ValidAddresses

	for _, invalidPassword := range invalidPasswords {

		_, err := NewDB(dbName, userName, invalidPassword, addresses)

		if err == nil {
			t.Errorf("NewDB did not return an error for password %s", invalidPassword)
		}
	}
}

// NewDB should return an error on empty or non-alpha list of addresses
func TestNewDB_ReturnsErrorOnInvalidAddress(t *testing.T) {
	invalidAddressesList := [][]string{
		{""},
		{"#@#$#@"},
		{"0123"},
	}

	dbName := ValidDBName
	userName := ValidUserName
	password := ValidPassword

	for _, invalidAddresses := range invalidAddressesList {

		_, err := NewDB(dbName, userName, password, invalidAddresses)

		if err == nil {
			t.Errorf("NewDB did not return an error for addresses %s", invalidAddresses)
		}
	}
}

// GetDBName should match expected value when created with valid inputs
func TestGetDBName_ReturnsExpectedValidValue(t *testing.T) {
	db := createValidDB(t)
	expectedDbName := ValidDBName

	if db.GetDBName() != expectedDbName {
		t.Errorf("GetDBName() did not return expected value of %s", expectedDbName)
	}
}

// GetUserName should match expected value when created with valid inputs
func TestGetUserName_ReturnsExpectedValidValue(t *testing.T) {
	db := createValidDB(t)
	expectedUserName := ValidUserName

	if db.GetUserName() != expectedUserName {
		t.Errorf("GetUserName() did not return expected value of %s", expectedUserName)
	}
}

// GetPassword should match expected value when created with valid inputs
func TestGetPassword_ReturnsExpectedValidValue(t *testing.T) {
	db := createValidDB(t)
	expectedPassword := ValidPassword

	if db.GetPassword() != expectedPassword {
		t.Errorf("GetPassword() did not return expected value of %s", expectedPassword)
	}
}

// GetAddresses should match expected values when created with valid inputs
func TestGetAddresses_ReturnsExpectedValidValue(t *testing.T) {
	db := createValidDB(t)
	expectedAddresses := ValidAddresses

	addresses := db.GetAddresses()
	for i, address := range addresses {
		if address != expectedAddresses[i] {
			t.Errorf("GetAddresses() did not return expected value of %s on %d",
				expectedAddresses, i)
		}
	}

}

// createValidDB is a helper test function
// which creates and returns a valid DB instance
func createValidDB(t *testing.T) DB {
	dbName := ValidDBName
	userName := ValidUserName
	password := ValidPassword
	addresses := ValidAddresses

	db, err := NewDB(dbName, userName, password, addresses)

	if err != nil {
		t.Error("NewDB received invalid inputs")
	}

	return db
}
