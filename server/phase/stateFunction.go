package phase

type IncrementState func(to State) bool

type GetState func() State
