////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package storage

import (
	"bytes"
	"reflect"
	"strings"
	"sync"
	"testing"
)

// Unit test
func TestNewNodeSecretManager(t *testing.T) {
	// Initialize manager
	testManager := NewNodeSecretManager()

	// Check that manager is initialized as empty
	if len(testManager.secrets) != 0 {
		t.Fatalf("Node maanger map should be empty.")
	}

	// Initialize an expected node secret initialization
	expected := &NodeSecretManager{
		secrets: make(map[int]*NodeSecret, MaxNodeSecrets),
		mux:     sync.Mutex{},
	}

	// Check that expected initialization state matches constructor
	if !reflect.DeepEqual(testManager, expected) {
		t.Fatalf("New maanger does not match expected output."+
			"\n\tExpected: %v"+
			"\n\tRecieved: %v", testManager, expected)
	}

}

// Happy path
func TestNodeSecretManager_UpsertSecret(t *testing.T) {
	// Initialize manager
	testManager := NewNodeSecretManager()

	// Construct secret and key id
	secret := []byte("test1234")
	keyId := 0

	// Insert into manager
	err := testManager.UpsertSecret(keyId, secret)
	if err != nil {
		t.Fatalf("UpsertSecret received an error: %v", err)
	}

	// Test that manager has been written to
	if len(testManager.secrets) != 1 {
		t.Fatalf("Node maanger map should contain one element.")
	}

	// Recreate secret using data passed to UpsertSecret
	expected := make([]byte, SecretSize)
	copy(expected, secret)

	// Check if entry in manager contains expected value
	val, ok := testManager.secrets[keyId]
	if !ok || !bytes.Equal(val.Secret.Bytes(), expected) {
		t.Fatalf("UpsertSecret did not upsert expected value in manager."+
			"\n\tExpected: %v"+
			"\n\tReceived: %v", expected, val.Secret.Bytes())
	}

}

// Happy path: Test that upserted value overwrites existing entry
func TestNodeSecretManager_UpsertSecret_Overwrite(t *testing.T) {
	// Initialize manager
	testManager := NewNodeSecretManager()

	// Construct secret and key id
	secret := []byte("test1234")
	keyId := 0

	// Insert into manager
	err := testManager.UpsertSecret(keyId, secret)
	if err != nil {
		t.Fatalf("UpsertSecret received an error: %v", err)
	}

	// Construct new value for overwriting
	newVal := []byte("newValue")
	err = testManager.UpsertSecret(keyId, newVal)
	if err != nil {
		t.Fatalf("UpsertSecret error: %v", err)
	}

	// Recreate secret using data passed to UpsertSecret
	newValExpected := make([]byte, SecretSize)
	copy(newValExpected, newVal)

	// Check that value is overwritten
	receivedVal := testManager.secrets[keyId].Secret
	if !bytes.Equal(newValExpected, receivedVal.Bytes()) {
		t.Fatalf("UpsertSecret did not overwrite existing entry."+
			"\n\tExpected: %v"+
			"\n\tReceived: %v", newValExpected, receivedVal.Bytes())
	}
}

// Error path: Test that a secret exceeding SecretSize cannot be inserted
func TestNodeSecretManager_UpsertSecret_BadSecret(t *testing.T) {
	// Initialize manager
	testManager := NewNodeSecretManager()

	// Construct secret with a bad length
	badSecret := make([]byte, SecretSize*2)
	copy(badSecret, "Too big")

	// Insert bad secret
	err := testManager.UpsertSecret(0, badSecret)

	// Check that expected error was encountered
	if err == nil || !strings.HasSuffix(err.Error(), BadSecretSizeError) {
		t.Fatalf("Expected error state: Should not be able to insert "+
			"secret with length greater than %d. "+
			"\n\tError expected: %v"+
			"\n\tError received: %v", SecretSize, BadSecretSizeError, err)
	}

}

// Error path: Attempt to insert secret when manager has MaxNodeSecrets inserted.
func TestNodeSecretManager_UpsertSecret_FullManager(t *testing.T) {
	// Initialize manager
	testManager := NewNodeSecretManager()

	// Fill up manager to MaxNodeSecrets
	for i := 0; i < MaxNodeSecrets; i++ {
		err := testManager.UpsertSecret(i, []byte("Filling up"))
		if err != nil {
			t.Fatalf("UpsertSecret unexpected error: %v", err)
		}
	}

	// Insert entry number MaxNodeSecrets + 1
	err := testManager.UpsertSecret(MaxNodeSecrets+1, []byte("One too many"))

	// Check that expected error was encountered
	if err == nil || !strings.HasSuffix(err.Error(), ManagerFullError) {
		t.Fatalf("Expected error state: Should not be able to insert "+
			"secret when manager is full. Manager size: %d "+
			"\n\tError expected: %v"+
			"\n\tError received: %v", len(testManager.secrets), ManagerFullError, err)
	}

}

// Happy path
func TestNodeSecretManager_GetSecret(t *testing.T) {
	// Initialize manager
	testManager := NewNodeSecretManager()

	// Construct secret and key id
	secret := []byte("test1234")
	keyId := 0

	// Insert secret
	err := testManager.UpsertSecret(keyId, secret)
	if err != nil {
		t.Fatalf("UpsertSecret error: %v", err)
	}

	received, err := testManager.GetSecret(keyId)
	if err != nil {
		t.Fatalf("GetSecret error: %v", err)
	}

	// Recreate secret using data passed to UpsertSecret
	expected := make([]byte, SecretSize)
	copy(expected, secret)

	// Check that GetSecret retrieved expected value
	if !bytes.Equal(received.Bytes(), expected) {
		t.Fatalf("GetSecret retrieved unexpected value."+
			"\n\tExpected: %v"+
			"\n\tReceived: %v", expected, received.Bytes())
	}
}

// Error path: Get an entry that does not exist
func TestNodeSecretManager_GetSecret_NoEntry(t *testing.T) {
	// Initialize an empty manager
	testManager := NewNodeSecretManager()

	// Attempt to retrieve non existent entry
	_, err := testManager.GetSecret(0)

	// Check that expected error received
	if err == nil || !strings.ContainsAny(err.Error(), NoSecretExistsError) {
		t.Fatalf("GetSecret expected error: Should error when requesting a non-existant entry."+
			"\n\tError expected: %v"+
			"\n\tError received: %v", err, NoSecretExistsError)
	}
}

// Happy path
func TestNodeSecretManager_delete(t *testing.T) {
	// Initialize manager
	testManager := NewNodeSecretManager()

	// Construct secret and key id
	secret := []byte("test1234")
	keyId := 0

	// Insert secret
	err := testManager.UpsertSecret(keyId, secret)
	if err != nil {
		t.Fatalf("UpsertSecret error: %v", err)
	}

	// Delete entry
	err = testManager.delete(keyId)
	if err != nil {
		t.Fatalf("delete error: %v", err)
	}

	// Check that expected error received
	_, err = testManager.GetSecret(keyId)
	if err == nil || !strings.ContainsAny(err.Error(), NoSecretExistsError) {
		t.Fatalf("GetSecret expected error: Should error when requesting a non-existant entry."+
			"\n\tError expected: %v"+
			"\n\tError received: %v", err, NoSecretExistsError)
	}

}

// Error path: delete a non existent entry
func TestNodeSecretManager_delete_NoEntry(t *testing.T) {
	// Initialize manager
	testManager := NewNodeSecretManager()

	// Delete non existent entry
	err := testManager.delete(0)

	// Check that expected error received
	if err == nil || !strings.ContainsAny(err.Error(), NoSecretExistsError) {
		t.Fatalf("GetSecret expected error: Should error when requesting a non-existant entry."+
			"\n\tError expected: %v"+
			"\n\tError received: %v", err, NoSecretExistsError)
	}
}

// Happy path
func TestNodeSecretManager_getNodeSecret(t *testing.T) {
	// Initialize manager
	testManager := NewNodeSecretManager()

	// Construct secret and key id
	secret := []byte("test1234")
	keyId := 0

	// Insert secret
	err := testManager.UpsertSecret(keyId, secret)
	if err != nil {
		t.Fatalf("UpsertSecret error: %v", err)
	}

	// Retrieve value from manager
	received, err := testManager.getNodeSecret(keyId)
	if err != nil {
		t.Fatalf("getNodeSecret error: %v", err)
	}

	// Construct expected value
	expectedData := Secret{}
	copy(expectedData[:], secret)
	expected := &NodeSecret{Secret: expectedData}

	// Check that retrieved value matches expected
	if !reflect.DeepEqual(expected, received) {
		t.Fatalf("getNodeSecret retrieved unexpected data."+
			"\n\tExpected: %v"+
			"\n\tReceived: %v", expected, received)
	}
}

// Error path: Retrieve node secret that is not in manager.
func TestNodeSecretManager_getNodeSecret_NonExistingEntry(t *testing.T) {
	// Initialize manager
	testManager := NewNodeSecretManager()

	// Retrieve value from manager
	_, err := testManager.getNodeSecret(0)
	if err == nil || !strings.ContainsAny(err.Error(), NoSecretExistsError) {
		t.Fatalf("getNodeSecret expected error: "+
			"Should error when requesting non existant entry."+
			"\n\tExpected error: %v"+
			"\n\tReceived error: %v", NoSecretExistsError, err)
	}

}
