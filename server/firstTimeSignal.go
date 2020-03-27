////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package server

// todo file description

import (
	"errors"
	"sync"
	"time"
)

type FirstTime struct {
	c chan struct{}
	sync.Once
}

// NewFirstTime is a constructor of the FirstTime object
func NewFirstTime() *FirstTime {
	return &FirstTime{
		c:    make(chan struct{}, 1),
		Once: sync.Once{},
	}
}

// Send sends to the structs channel explicitly once
func (ft *FirstTime) Send() {
	ft.Once.Do(func() {
		ft.c <- struct{}{}
	})
}

// Receive either receives from the channel or times out
// On timeout it errors.
func (ft *FirstTime) Receive(duration time.Duration) error {

	select {
	case <-ft.c:
		return nil
	case <-time.After(duration):
		return errors.New("Timed out receiving from first time channel")
	}
}
