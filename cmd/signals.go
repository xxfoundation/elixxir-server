///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package cmd

import (
	jww "github.com/spf13/jwalterweatherman"
	"os"
	"os/signal"
	"syscall"
)

// signals.go handles signals specific to the permissioning server:
//   - SIGUSR1, which stops round creation
//   - SIGTERM/SIGINT, which stops round creation and exits
//
// The functions are set up to receive arbitrary functions that handle
// the necessary behaviors instead of implementing the behavior directly.

// ReceiveSignal calls the provided function when it receives a specific
// signal. It will call the provided function every time it recieves the signal.
func ReceiveSignal(sigFn func(), sig os.Signal) {
	// Set up channel on which to send signal notifications.
	// We must use a buffered channel or risk missing the signal
	// if we're not ready to receive when the signal is sent.
	c := make(chan os.Signal, 1)
	signal.Notify(c, sig)

	// Block until a signal is received, then call the function
	// provided
	go func() {
		for {
			<-c
			jww.INFO.Printf("Received %s signal...\n", sig)
			sigFn()
		}
	}()
}

// ReceiveUSR1Signal calls the provided function when receiving SIGUSR1.
// It will call the provided function every time it receives it
func ReceiveUSR1Signal(usr1Fn func()) {
	ReceiveSignal(usr1Fn, syscall.SIGUSR1)
}

// ReceiveUSR2Signal calls the provided function when receiving SIGUSR1.
// It will call the provided function every time it receives it
func ReceiveUSR2Signal(usr1Fn func()) {
	ReceiveSignal(usr1Fn, syscall.SIGUSR2)
}

// ReceiveExitSignal signals a stop chan when it receives
// SIGTERM or SIGINT
func ReceiveExitSignal() chan os.Signal {
	// Set up channel on which to send signal notifications.
	// We must use a buffered channel or risk missing the signal
	// if we're not ready to receive when the signal is sent.
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	return c
}
