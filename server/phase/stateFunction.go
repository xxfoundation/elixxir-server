package phase

//IncrementState is used to move the state forward for a phase.  If the incrementation fails, it returns false
type IncrementState func(to State) bool

type GetState func() State
