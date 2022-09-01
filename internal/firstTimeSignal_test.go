////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package internal

import (
	"testing"
	"time"
)

//tests that sending works only on the first try
func TestFirstTime_Send(t *testing.T) {
	ft := NewFirstTime()

	//send the first time
	ft.Send()

	select {
	case <-ft.c:
	case <-time.After(time.Millisecond):
		t.Errorf("First time send did not occur")
	}

	// send the second time
	ft.Send()

	select {
	case <-ft.c:
		t.Errorf("send should not occur on second use")
	case <-time.After(time.Millisecond):
	}
}

//tests that receiving works and waits until the send occurs

func TestFirstTime_Receive(t *testing.T) {
	ft := NewFirstTime()

	received := make(chan bool)

	go func() {
		ft.Receive()
		received <- true
	}()

	select {
	case <-time.After(50 * time.Millisecond):
	case <-received:
		t.Errorf("receive should not have happened, send has not called")
	}

	ft.c <- struct{}{}

	select {
	case <-time.After(50 * time.Millisecond):
		t.Errorf("receive after send timed out")
	case <-received:
	}
}
