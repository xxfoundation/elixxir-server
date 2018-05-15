package io

import (
	"testing"
	"time"
)

func TestVerifyServersOnline(t *testing.T) {
	done := 0
	servers := [1]string{"localhost:5555"}
	go func(d *int) {
		time.Sleep(2 * time.Second)
		*d = 1
	}(&done)
	VerifyServersOnline(servers[:])
	if done == 1 {
		t.Errorf("Could not verify servers in less than 2 seconds!")
	}
}
