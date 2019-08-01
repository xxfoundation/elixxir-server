package phase

// Transition is used to move between states in a phase.
// The transition can fail because the connection between
//IncrementState is used to move the state forward for a phase.
// If the incrementation fails, it returns false
type Transition func(from, to State) bool

type GetState func() State
