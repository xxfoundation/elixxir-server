////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package measure

// measure_Tags.go contains the string constants for our measure tags

// Constants for Tag strings used by Measure()
const (
	TagReceiveOnReception = "Receive Header/Edge Checks"
	TagActive             = "Active"
	TagTransmitter        = "Transmit Header/Start Transmitter"
	TagReceiveFirstSlot   = "Receive First Slot"
	TagFinishFirstSlot    = "Finish First Slot"
	TagReceiveLastSlot    = "Receive Last Slot"
	TagFinishLastSlot     = "Finish Last Slot"
	TagTransmitLastSlot   = "Transmit Last Slot"
	TagVerification       = "Verification"
)
