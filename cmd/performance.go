////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package cmd

import (
	"bufio"
	"errors"
	"fmt"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/server/server/measure"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	// Amount of memory allocation required before the system triggers a
	// performance alert
	deltaMemoryThreshold = uint64(1024) * uint64(1024) * uint64(100)  // 100MiB
	minMemoryTrigger     = uint64(1024) * uint64(1024) * uint64(1024) // 1GiB

	// Time between performance checks
	performanceCheckPeriod = 15 * time.Second

	// Specifies the number of recently executed functions to display
	numFuncPrint = 10

	// Base for CPU usage numbers
	cpuUsageBase = 10

	// Bit size for converting CPU usage number strings to integers
	cpuUsageBitSize = 64
)

// MonitorMemoryUsage checks and prints a warning every time thread or memory
// usage fo the system jumps a designated amount.
func monitorMemoryUsage(perfCheck time.Duration, deltaMem,
	minMem uint64) *measure.ResourceMonitor {

	systemStartTime := time.Now()
	lastTrigger := time.Now()

	resourceMetric := measure.ResourceMetric{
		SystemStartTime: systemStartTime,
		Time:            lastTrigger,
		MemAllocBytes:   0,
		MemAvailable:    0,
		NumThreads:      0,
		CPUPercentage:   0.0,
	}
	resourceMonitor := measure.ResourceMonitor{}
	resourceMonitor.Set(&resourceMetric)

	cpu := cpuMeasure{}
	_, _ = cpu.getCPUUsage()

	go func() {
		var pastMemoryAllocated uint64

		for {
			triggerTime := time.Now()

			// Get amount of memory allocated, in bytes
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			memoryAllocated := m.Alloc
			memoryAvailable := m.Sys

			// Get the number of executing goroutines
			currentThreads := runtime.NumGoroutine()

			// Get CPU usage percentage
			cpuPercentage, err := cpu.getCPUUsage()
			if err != nil {
				jww.WARN.Printf("Could not get CPU usage info: %v", err)
			} else {
				jww.INFO.Printf("Got CPU usage: %f", cpuPercentage)
			}

			// Save metric information
			resourceMetric = measure.ResourceMetric{
				SystemStartTime: systemStartTime,
				Time:            triggerTime,
				MemAllocBytes:   memoryAllocated,
				MemAvailable:    memoryAvailable,
				NumThreads:      currentThreads,
				CPUPercentage:   cpuPercentage,
			}
			resourceMonitor.Set(&resourceMetric)

			// Calculate information on when to trigger prints
			deltaTriggerTime := triggerTime.Sub(lastTrigger)
			memoryDelta := memoryAllocated - pastMemoryAllocated
			pastMemoryAllocated = memoryAllocated

			// Check if the change in memory usage warrants an update
			if memoryDelta >= deltaMem && memoryAllocated >= minMem {
				lastTrigger = triggerTime

				jww.WARN.Printf("Performance warning triggered after %v "+
					"seconds", deltaTriggerTime*time.Second)

				jww.WARN.Printf("  Allocated Memory %v exceeded threshold of %v",
					convertToReadableBytes(memoryAllocated), pastMemoryAllocated)

				jww.WARN.Printf("  Number of threads: %v", currentThreads)
				jww.WARN.Printf("  Top %d threads by memory allocation:",
					numFuncPrint)
			}

			// Only trigger periodically
			time.Sleep(perfCheck)
		}
	}()

	return &resourceMonitor
}

// Nice is when the CPU is executing a user task at below-normal priority.
// guest and guestNice are already accounted in user and nice and are thus not included in the total calculation.
type cpuMeasure struct {
	// Components of the total CPU ticks since boot
	user      uint64 // Normal processes executing in user mode
	nice      uint64 // Niced processes executing in user mode
	system    uint64 // Processes executing in kernel mode
	idle      uint64 // No processes being run
	ioWait    uint64 // Waiting for I/O to complete
	irq       uint64 // Servicing interrupt requests (IRQs)
	softIRQ   uint64 // Servicing soft IRQs
	steal     uint64 // Involuntary wait
	guest     uint64 // Running a normal guest
	guestNice uint64 // Running a niced guest

	used   uint64 // Total CPU time since boot
	unused uint64 // Total CPU Idle time since boot
	total  uint64 // Total CPU usage time since boot
}

// GetCPUUsage
func (c *cpuMeasure) getCPUUsage() (percent float64, err error) {
	// Open /proc/stat
	file, err := os.Open("/proc/stat")
	if err != nil {
		return 0, err
	}

	// Read our file line by line (we only want the first line anyway)
	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)
	scanner.Scan()

	// Example of first line: "cpu  1479 291 2510 5620 117 0 20 0 0 0"
	text := scanner.Text()

	// Remove first five characters "cpu  "
	text = text[5:]

	// Split result into parts
	parts := strings.Fields(text)

	err = file.Close()
	if err != nil {
		return 0, err
	}

	return c.parseCPUUsage(parts)
}

// parseCPUUsage returns the percent utilisation of the entire CPU (all cores
// summed) since the last time the function was called. CPU usage is calculated
// by dividing the number of "ticks" a CPU is active by the total number of
// "ticks" over the time is was observed. Since that period is time is based on
// the previous call, the first time this function is run, the output will be
// garbage.
func (c *cpuMeasure) parseCPUUsage(parts []string) (float64, error) {
	// Check that all 10 components of CPU utilisation are passed in
	if len(parts) != 10 {
		return 0, errors.New("expected 10 CPU components in the input array")
	}

	// Construct a new object to store CPU information
	var err error
	cpu := cpuMeasure{}

	// Convert string values to integers
	cpu.user, err = strconv.ParseUint(parts[0], cpuUsageBase, cpuUsageBitSize)
	if err != nil {
		return 0, err
	}

	cpu.nice, err = strconv.ParseUint(parts[1], cpuUsageBase, cpuUsageBitSize)
	if err != nil {
		return 0, err
	}

	cpu.system, err = strconv.ParseUint(parts[2], cpuUsageBase, cpuUsageBitSize)
	if err != nil {
		return 0, err
	}

	cpu.idle, err = strconv.ParseUint(parts[3], cpuUsageBase, cpuUsageBitSize)
	if err != nil {
		return 0, err
	}

	cpu.ioWait, err = strconv.ParseUint(parts[4], cpuUsageBase, cpuUsageBitSize)
	if err != nil {
		return 0, err
	}

	cpu.irq, err = strconv.ParseUint(parts[5], cpuUsageBase, cpuUsageBitSize)
	if err != nil {
		return 0, err
	}

	cpu.softIRQ, err = strconv.ParseUint(parts[6], cpuUsageBase, cpuUsageBitSize)
	if err != nil {
		return 0, err
	}

	cpu.steal, err = strconv.ParseUint(parts[7], cpuUsageBase, cpuUsageBitSize)
	if err != nil {
		return 0, err
	}

	cpu.guest, err = strconv.ParseUint(parts[8], cpuUsageBase, cpuUsageBitSize)
	if err != nil {
		return 0, err
	}

	cpu.guestNice, err = strconv.ParseUint(parts[9], cpuUsageBase, cpuUsageBitSize)
	if err != nil {
		return 0, err
	}

	// Calculate total CPU time since boot
	cpu.total = cpu.user + cpu.nice + cpu.system + cpu.idle + cpu.ioWait +
		cpu.irq + cpu.softIRQ + cpu.steal

	// Calculate total CPU idle time since boot
	cpu.unused = cpu.idle + cpu.ioWait

	// Calculate total CPU usage time since boot
	cpu.used = cpu.total - cpu.unused

	// Calculate total CPU time and total CPU usage time since last check
	deltaTotalTime := cpu.total - c.total
	deltaTotalUsed := cpu.used - c.used

	// Check that CPU time has progressed since last function call (to avoid
	// division by zero)
	if deltaTotalTime == 0 {
		*c = cpu
		return 0, errors.New("no CPU time progression since last call")
	}

	// Calculate total CPU percentage
	percent := (float64(deltaTotalUsed) / float64(deltaTotalTime)) * 100.0

	// Save new CPU information
	*c = cpu

	return percent, nil
}

// convertToReadableBytes converts the given number of bytes into a readable
// data size with units.
func convertToReadableBytes(b uint64) string {
	const unit = 1024

	if b < unit {
		return fmt.Sprintf("%d B", b)
	}

	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}
