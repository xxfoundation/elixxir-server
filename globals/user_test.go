///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package globals

import (
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/xx_network/crypto/large"
	"gitlab.com/xx_network/primitives/id"
	"testing"
)

const numTestDemoUsers = 256

// TestUserRegistry tests the constructors/getters/setters
// surrounding the User struct and the UserRegistry interface
// TODO: This test needs split up
func TestUserRegistry(t *testing.T) {
	grp := InitCrypto()

	users := UserRegistry(&UserMap{})

	for i := 0; i < numTestDemoUsers; i++ {
		u := users.NewUser(grp)
		users.UpsertUser(u)
	}

	// TESTS Start here

	numUsers := users.CountUsers()

	if numUsers != numTestDemoUsers {
		t.Errorf("Count users is not working correctly. "+
			"Expected: %v Actual: %v", numTestDemoUsers, numUsers)
	}

	id9 := id.NewIdFromUInt(9, id.User, t)
	usr9, err := users.GetUser(id9, grp)

	if err != nil {
		t.Errorf("User fetch returned error: %s", err.Error())
	}

	if usr9 == nil {
		t.Fatalf("Error fetching user!")
	}

	getUser, err := users.GetUser(usr9.ID, grp)

	if (err != nil) || getUser.ID != usr9.ID {
		t.Errorf("GetUser: Returned unexpected result for user lookup!")
	}

	usr3, _ := users.GetUser(id.NewIdFromUInt(3, id.User, t), grp)
	usr5, _ := users.GetUser(id.NewIdFromUInt(5, id.User, t), grp)

	if usr3.BaseKey == usr5.BaseKey {
		t.Errorf("Transmissions keys are the same and they should be different!")
	}

	users.DeleteUser(usr9.ID)

	if users.CountUsers() != numTestDemoUsers-1 {
		t.Errorf("User has not been deleted correctly. "+
			"Expected # of users: %v Actual: %v", numTestDemoUsers-1, users.CountUsers())
	}

	if _, userExists := users.GetUser(usr9.ID, grp); userExists == nil {
		t.Errorf("DeleteUser: Excepted zero value for deleted user lookup!")
	}
}

// Test happy path
func TestUser_DeepCopy(t *testing.T) {
	grp := InitCrypto()

	users := UserRegistry(&UserMap{})

	user := users.NewUser(grp)
	user.BaseKey = grp.NewInt(66)

	newUser := user.DeepCopy()
	if user.BaseKey.Cmp(newUser.BaseKey) != 0 {
		t.Errorf("User Deepcopy: Failed to copy keys!")
	}

	var uNil *User

	uNilCpy := uNil.DeepCopy()

	if uNilCpy != nil {
		t.Errorf("User Deepcopy: copy occured on nil user")
	}
}

// Test happy path and inserting too many salts
func TestUserMap_InsertSalt(t *testing.T) {
	grp := InitCrypto()

	users := UserRegistry(&UserMap{})
	u9 := users.NewUser(grp)
	u9.ID = id.NewIdFromUInt(1, id.User, t)
	users.UpsertUser(u9)

	// Insert like 300 salts, expect success
	for i := 0; i < MaxSalts; i++ {
		err := users.InsertSalt(u9.ID, []byte("test"))
		if err != nil {
			t.Errorf("InsertSalt: Expected success! Received: %s", err.Error())
		}
	}
	// Now we have exceeded the max number, expect failure
	err := users.InsertSalt(u9.ID, []byte("test"))
	if err == nil {
		t.Errorf("InsertSalt: Expected failure due to exceeding max count of" +
			" salts for one user, received success")
	}
}

// Tests that InsertSalt() returns the error ErrNonexistantUser when the given
// user ID is not in the user map.
func TestUserMap_InsertSalt_ErrNonexistantUser(t *testing.T) {
	grp := InitCrypto()

	users := UserRegistry(&UserMap{})
	u9 := users.NewUser(grp)
	u9.ID = id.NewIdFromUInt(1, id.User, t)

	err := users.InsertSalt(id.NewIdFromUInt(2, id.User, t), []byte("test"))
	if err != ErrNonexistantUser {
		t.Errorf("InsertSalt: Expected error when using User ID that does not "+
			"exist.\n\texpected: %v\n\treceived: %v", ErrNonexistantUser, err)
	}
}

// InitCrypto sets up the cryptographic constants for cMix
func InitCrypto() *cyclic.Group {

	base := 16
	// FIXME: Which prime is this? Comment and link the appropriate source!
	// This doesn't even look prime...
	pString := "9DB6FB5951B66BB6FE1E140F1D2CE5502374161FD6538DF1648218642F0B5C48" +
		"C8F7A41AADFA187324B87674FA1822B00F1ECF8136943D7C55757264E5A1A44F" +
		"FE012E9936E00C1D3E9310B01C7D179805D3058B2A9F4BB6F9716BFE6117C6B5" +
		"B3CC4D9BE341104AD4A80AD6C94E005F4B993E14F091EB51743BF33050C38DE2" +
		"35567E1B34C3D6A5C0CEAA1A0F368213C3D19843D0B4B09DCB9FC72D39C8DE41" +
		"F1BF14D4BB4563CA28371621CAD3324B6A2D392145BEBFAC748805236F5CA2FE" +
		"92B871CD8F9C36D3292B5509CA8CAA77A2ADFC7BFD77DDA6F71125A7456FEA15" +
		"3E433256A2261C6A06ED3693797E7995FAD5AABBCFBE3EDA2741E375404AE25B"
	// FIXME: Or deleteme, this can't be a generator!
	gString := "5C7FF6B06F8F143FE8288433493E4769C4D988ACE5BE25A0E24809670716C613" +
		"D7B0CEE6932F8FAA7C44D2CB24523DA53FBE4F6EC3595892D1AA58C4328A06C4" +
		"6A15662E7EAA703A1DECF8BBB2D05DBE2EB956C142A338661D10461C0D135472" +
		"085057F3494309FFA73C611F78B32ADBB5740C361C9F35BE90997DB2014E2EF5" +
		"AA61782F52ABEB8BD6432C4DD097BC5423B285DAFB60DC364E8161F4A2A35ACA" +
		"3A10B1C4D203CC76A470A33AFDCBDD92959859ABD8B56E1725252D78EAC66E71" +
		"BA9AE3F1DD2487199874393CD4D832186800654760E1E34C09E4D155179F9EC0" +
		"DC4473F996BDCE6EED1CABED8B6F116F7AD9CF505DF0F998E34AB27514B0FFE7"

	p := large.NewIntFromString(pString, base)
	g := large.NewIntFromString(gString, base)

	grp := cyclic.NewGroup(p, g)

	return grp
}
