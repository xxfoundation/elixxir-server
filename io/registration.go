////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Handles creating callbacks for registration hooks into comms library

package io

// Handle nonce request from Client
func (s ServerImpl) RequestNonce(salt, diffieKey, Y, P, Q, G,
	hash, R, S []byte) ([]byte, error) {

	// Verify signed public key using hardcoded RegistrationServer public key
	// If valid signed public key && public key is unique:
	//     Generate UserID by hashing salt and Client public key
	//     Store UserID, Client public key,
	//       and Diffie Hellman key in user database
	//     Generate and store a nonce (with a TTL) in user database
	//     Return nonce to Client with empty error field

	// If invalid signed public key || not unique:
	//     Return empty nonce to Client with relevant error

	return make([]byte, 0), nil
}

// Handle confirmation of nonce from Client
func (s ServerImpl) ConfirmNonce(hash, R,
	S []byte) ([]byte, []byte, []byte, error) {

	// Verify signed nonce using Client public key (from Step 7a),
	//  ensuring TTL has not expired
	// If valid signed nonce:
	//     Update user database entry to indicate successful registration
	//     Use hardcoded Server keypair to sign Client public key
	//     Return signed Client public key to Client with empty error field

	// If invalid signed nonce:
	//     Return empty public key to Client with relevant error

	return make([]byte, 0), make([]byte, 0), make([]byte, 0), nil
}
