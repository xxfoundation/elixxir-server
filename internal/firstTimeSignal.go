////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package internal

// firstTimeSignal.go contains the logic for a channel
// that can only be sent to once

import (
	"fmt"
	jww "github.com/spf13/jwalterweatherman"
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

// Receive either receives from the channel. Prints a log ever `duration` to
// notify it is still waiting
func (ft *FirstTime) Receive(duration time.Duration, reason string)  {
	logMessage := fmt.Sprintf("Waiting on %s to continue", reason)
	jww.INFO.Printf(logMessage)
	ticker := time.NewTicker(duration)
	for true{
		select {
		case <-ft.c:
			return
		case <-ticker.C:
			jww.WARN.Printf(logMessage)
		}
	}

}
