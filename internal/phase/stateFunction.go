///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package phase

// Transition is used to move between states in a phase.
// The transition can fail because the connection between
//IncrementState is used to move the state forward for a phase.
// If the incrementation fails, it returns false
type Transition func(from, to State) bool

type GetState func() State
