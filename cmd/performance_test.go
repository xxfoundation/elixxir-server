////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package cmd

import (
	"testing"
	"time"
)

func TestMonitorMemoryUsage(t *testing.T) {
	perfCheckPeriod := time.Duration(1) * time.Second
	deltaMem := uint64(0)
	minMem := uint64(0)

	rm := monitorMemoryUsage(perfCheckPeriod, deltaMem, minMem)

	// rm := monitorMemoryUsage(performanceCheckPeriod, deltaMemoryThreshold, minMemoryTrigger)

	resourceMetric := rm.Get()

	t0 := resourceMetric.Time

	time.Sleep(performanceCheckPeriod * time.Duration(2))

	resourceMetric = rm.Get()
	t1 := resourceMetric.Time

	if !t1.After(t0) {
		t.Errorf("Resource metric not being updated in MonitorMemoryUsage")
	}
}

// Test the CPU parsing function
func TestCPUParsing(t *testing.T) {
	cpu := cpuMeasure{}

	// test less than 10 items
	test := []string{"6", "9"}
	_, e := cpu.parseCPUUsage(test)
	if e == nil {
		t.Errorf("parseCPUUsage did not return an error with junk inputs, (%s)", e)
	}

	// test garbage data
	test = []string{"9", "9", "9", "a", "a", "9", "9", "9", "9", "9"}
	_, e = cpu.parseCPUUsage(test)
	if e == nil {
		t.Errorf("parseCPUUsage did not return an error with junk inputs, (%s)", e)
	}

	// input first set of numbers (init function)
	// The function should be called on startup in the server to initalise the
	// data array, since the function works by getting the average CPU usage
	// since the last call. This is simulating the first call on startup.
	test = []string{"2726", "782", "1117", "4684", "108", "0", "51", "0", "5", "5"}
	r, e := cpu.parseCPUUsage(test)
	if r != 49.38741022391213 {
		t.Errorf("parseCPUUsage returned incorrect percent"+
			"\n\texpected: %f\n\treceived: %f", 49.38741022391213, r)
	}
	if e != nil {
		t.Error(e)
	}

	// our test of averages
	test = []string{"2738", "782", "1120", "6045", "108", "0", "51", "1", "0", "0"}
	r, e = cpu.parseCPUUsage(test)
	if r != 1.1619462599854757 {
		t.Errorf("parseCPUUsage returned incorrect percent"+
			"\n\texpected: %f\n\treceived: %f", 1.1619462599854757, r)
	}
	if e != nil {
		t.Error(e)
	}
}

// Tests that convertToReadableBytes() properly converts the number and appends
// the correct unit.
func Test_ConvertToReadableBytes(t *testing.T) {
	// Setup two arrays with the numbers to be converted and their converted
	// string
	byteArr := []uint64{
		7840,
		455445443592708308,
		71438608,
		203628,
		20572734,
		7,
		8768,
		13500310316,
		6757059270592,
		05360764,
		0,
	}
	expectArr := []string{
		"7.7 KiB",
		"404.5 PiB",
		"68.1 MiB",
		"198.9 KiB",
		"19.6 MiB",
		"7 B",
		"8.6 KiB",
		"12.6 GiB",
		"6.1 TiB",
		"1.4 MiB",
		"0 B",
	}

	// Convert the number and check convertToReadableBytes()'s output
	for i, val := range byteArr {
		str := convertToReadableBytes(val)

		if str != expectArr[i] {
			t.Errorf("convertToReadableBytes() did not properly convert "+
				"%d to a readable bytes string\n\texpected: %s\n\treceived: %s",
				val, expectArr[i], str)
		}
	}
}
