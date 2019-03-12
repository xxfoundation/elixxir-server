////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package io

import (
	"gitlab.com/elixxir/crypto/nonce"
	"gitlab.com/elixxir/server/globals"
	"testing"
)

// Test invalid client signature
func TestConfirmNonce_BadSignature(t *testing.T) {
	globals.Users = globals.NewUserRegistry("", "", "", "")

	user := globals.Users.NewUser()
	user.Nonce = nonce.NewNonce(nonce.RegistrationTTL)
	globals.Users.UpsertUser(user)

	_, _, _, err := ConfirmNonce(user.Nonce.Bytes(), make([]byte, 0),
		make([]byte, 0))
	if err == nil {
		t.Errorf("ConfirmNonce: Expected bad signature!")
	}
}
