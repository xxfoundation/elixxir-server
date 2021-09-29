///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package storage

import (
	"reflect"
	"sync"
	"testing"
)

// Unit test
func TestNewNodeSecretManager(t *testing.T) {
	testManager := NewNodeSecretManager()
	if len(testManager.secrets) != 0 {
		t.Fatalf("Node maanger map should be empty.")
	}

	expected := &NodeSecretManager{
		secrets: make(map[int]NodeSecret, MaxNodeSecrets),
		mux:     sync.Mutex{},
	}

	if !reflect.DeepEqual(testManager, expected) {
		t.Fatalf("New maanger does not match expected output."+
			"\n\tExpected: %v"+
			"\n\tRecieved: %v", testManager, expected)
	}

}

// Happy path
func TestNodeSecretManager_UpsertSecret(t *testing.T) {
	testManager := NewNodeSecretManager()

	secret := Secret{}
	copy(secret[:], "test123")

	keyId := 0
	nodeSecret := NodeSecret{Secret: secret}
	err := testManager.UpsertSecret(keyId, nodeSecret)
	if err != nil {
		t.Fatalf("UpsertSecret received an error: %v", err)
	}

	if len(testManager.secrets) != 1 {
		t.Fatalf("Node maanger map should contain one element.")
	}

}
