package cmd

import (
	"fmt"
	jww "github.com/spf13/jwalterweatherman"
	"runtime"
	"time"
)

// Amount of memory allocation required before the system triggers a
// performance alert
const DELTA_MEMORY_THREASHOLD = int64(100000000)

// Time between performance checks
const PERFORMANCE_CHECK_PERIOD = time.Duration(2) * time.Minute

// Checks and prints a warning every time thread or memory usage fo the system
// jumps a designated amount
func MonitorMemoryUsage() {

	defer func() {
		if r := recover(); r != nil {
			jww.ERROR.Printf("Performance monitoring failed due to errors"+
				": %v", r)
		} else {
			jww.ERROR.Printf("Performance monitoring failed unexpectedly")
		}
	}()

	var numMemory = int64(0)

	var lastTrigger = time.Now()

	//Null profile record for comparison
	minMemoryUse := runtime.MemProfileRecord{0, 0, 0, 0, nil}

	for {
		//Only trigger preiodicly
		time.Sleep(PERFORMANCE_CHECK_PERIOD)

		triggerTime := time.Now()
		deltaTriggerTime := triggerTime.Sub(lastTrigger)
		var pr []runtime.MemProfileRecord

		currentThreads := runtime.NumGoroutine()

		//Get the memory usage of all threads
		runtime.MemProfile(pr, true)

		memoryAllocated := int64(0)

		//Make sure that if there are too few records the system still functions
		numMaxRecords := 10
		if len(pr) < numMaxRecords {
			numMaxRecords = len(pr)
		}

		highestMemUsage := make([]*runtime.MemProfileRecord, numMaxRecords)

		//Clear the memory profile record slice
		for i := 0; i < len(highestMemUsage); i++ {
			highestMemUsage[i] = &minMemoryUse
		}

		//Find total allocated memory and top memory usage threads
		for i := 0; i < len(pr); i++ {
			memoryAllocated += pr[i].InUseBytes()

			for j := len(highestMemUsage) - 1; j > -1; j-- {
				if pr[i].InUseBytes() > highestMemUsage[j].InUseBytes() {
					highestMemUsage[j] = &pr[i]
					break
				}
			}
		}

		memoryDelta := memoryAllocated - numMemory

		//check if the change in memory usage warrants an update
		if memoryDelta > DELTA_MEMORY_THREASHOLD {

			lastTrigger = triggerTime

			jww.WARN.Printf("Performance warning triggered after "+
				"%v seconds", deltaTriggerTime*time.Second)

			jww.WARN.Printf("Allocated Memory %v exceeded threshold of %v"+
				convertToReadableBytes(memoryAllocated), numMemory)

			jww.WARN.Printf("Number of threads: %v", currentThreads)
			jww.WARN.Printf("Top 10 threads by memory allocation:")
			//Format the data from the top 10 threads for printing
			for _, thr := range highestMemUsage {

				//Get a list of the last 10 executed functions
				var funcNames string
				lenlookup := len(thr.Stack0)
				if lenlookup > 10 {
					lenlookup = 10
				}
				for i := 0; i < lenlookup; i++ {
					funcNames += trncateFuncName(runtime.FuncForPC(thr.
						Stack0[i]).Name())
				}

				//Print thread information
				jww.WARN.Printf("  %s %s", convertToReadableBytes(thr.
					InUseBytes()), funcNames)
			}
			numMemory = memoryAllocated
		}

	}

}

func trncateFuncName(name string) string {
	if len(name) > 11 {
		return name[0:7] + "..., "
	}
	return name + ", "
}

var sizeLookup = []string{"B", "KiB", "MiB", "GiB"}

func convertToReadableBytes(b int64) string {

	for i := 0; i < len(sizeLookup)-1; i++ {
		if b < 1024 {
			return fmt.Sprintf("%v%v", b, sizeLookup[i])
		}
		b = b / 1024
	}

	return fmt.Sprintf("%v%v", b, sizeLookup[len(sizeLookup)-1])

}
