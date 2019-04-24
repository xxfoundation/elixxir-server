package server

type PhaseState uint32

const (
	//Initialized: Data structures for the phase have been created but it is not ready to run
	Initialized PhaseState = iota
	//Available: Next phase to run according to round but no input has been received
	Available
	//Queued: Next phase to run according to round and input has been received but it
	// has not begun execution by resource manager
	Queued
	//Running: Next phase to run according to round and input has been received and it
	// is being executed by resource manager
	Running
	//Complete: Phase is complete
	Completed
)
