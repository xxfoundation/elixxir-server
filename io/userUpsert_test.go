////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package io

import (
	"gitlab.com/privategrity/crypto/id"
	"testing"
	"time"
)

func TestUserUpsertBroadcast(t *testing.T) {
	done := 0
	Servers = []string{"localhost:5555"}
	go func(d *int) {
		time.Sleep(2 * time.Second)
		*d = 1
	}(&done)
	UserUpsertBroadcast(id.ZeroID, make([]byte, 0))
	if done == 1 {
		t.Errorf("Could not broadcast upsert in less than 2 seconds!")
	}
}
