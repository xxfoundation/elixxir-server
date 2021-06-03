///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package node

import (
	"bytes"
	"encoding/json"
	"gitlab.com/elixxir/server/internal"
	"gitlab.com/elixxir/server/internal/measure"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/utils"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

// This will error when the type does not match the function
var _ internal.MetricsHandler = func(instance *internal.Instance, roundID id.Round) error {
	return GatherMetrics(instance, roundID)
}

// Tests  that buildMetricJSON marshals data correctly by unmarshalling the JSON
// it outputs and comparing it to the original structure.
func Test_buildMetricJSON(t *testing.T) {
	roundMetrics := []measure.RoundMetrics{
		measure.NewRoundMetrics(id.Round(10), 5),
		measure.NewRoundMetrics(id.Round(20), 15),
		measure.NewRoundMetrics(id.Round(30), 25),
	}

	var rmTest []measure.RoundMetrics

	data, err := buildMetricJSON(roundMetrics, false)

	if err != nil {
		t.Errorf("buildMetricJSON() unexpectedly returned an error: %v", err)
	}

	err = json.Unmarshal(data, &rmTest)

	if err != nil {
		t.Errorf("buildMetricJSON() created JSON data that could not be "+
			"unmarshalled: %v", err)
	}

	if !compareRoundMetrics(rmTest, roundMetrics) {
		t.Errorf("buildMetricJSON() did not correctly marshal JSON data"+
			"\n\texpected: %#v\n\treceived: %#v", roundMetrics, rmTest)
	}
}

func compareRoundMetrics(a, b []measure.RoundMetrics) bool {
	for i := range a {
		if a[i].NodeID != b[i].NodeID {
			return false
		} else if a[i].NumNodes != b[i].NumNodes {
			return false
		} else if a[i].Index != b[i].Index {
			return false
		} else if a[i].RoundID != b[i].RoundID {
			return false
		} else if !reflect.DeepEqual(a[i].PhaseMetrics, b[i].PhaseMetrics) {
			return false
		} else if !reflect.DeepEqual(a[i].ResourceMetric, b[i].ResourceMetric) {
			return false
		}
	}

	return true
}

// Tests that saveMetricJSON() creates a log file with the correct name with the
// correct content.
func Test_saveMetricJSON(t *testing.T) {
	data := []byte("test")
	filePathPre := "test-"
	filePathPost := ".log"
	roundID := id.Round(50)
	filePath1 := filePathPre + "*" + filePathPost
	filePath2 := filePathPre + strconv.FormatUint(uint64(roundID), 10) + filePathPost

	err := saveMetricJSON(data, filePath1, roundID)

	// Check if the file exists with the correct name
	if _, err = os.Stat(filePath2); os.IsNotExist(err) {
		t.Errorf("File %s created by saveMetricJSON() could not be found",
			filePath2)
	}

	// Check if the data written to the file is correct
	if fileData, _ := utils.ReadFile(filePath2); !bytes.Equal(fileData, data) {
		t.Errorf("Data written by saveMetricJSON() incorrect"+
			"\n\texpected: %s\n\treceived: %s", string(data), string(fileData))
	}

	// Remove the created log file
	_ = os.Remove(filePath2)
}

// Tests that saveMetricJSON() errors on invalid path.
func Test_saveMetricJSON_ErrorPath(t *testing.T) {
	data := []byte("test")
	filePathPre := "~a/test-"
	filePathPost := ".log"
	roundID := id.Round(50)
	filePath1 := filePathPre + "*" + filePathPost
	filePath2 := filePathPre + strconv.FormatUint(uint64(roundID), 10) + filePathPost

	err := saveMetricJSON(data, filePath1, roundID)

	// Check if the data written to the file is correct
	if err == nil {
		t.Errorf("saveMetricJSON() did not error on incorrect path")
	}

	// Remove the created log file
	_ = os.Remove(filePath2)
}

// Tests that ClearMetricsLogs() correctly deletes all the files matching the
// test file.
func TestClearMetricsLogs(t *testing.T) {
	testFile := "test/temp-*.txt"

	// Create test files to delete.
	err := createTempFiles(testFile, 3)
	if err != nil {
		t.Errorf("Unexpected error creating temporary files: %v", err)
	}

	// Delete the files
	err = ClearMetricsLogs(testFile)
	if err != nil {
		t.Errorf("Unexpected error when running ClearMetricsLogs(): %v", err)
	}

	// Get list of any files left that match the test file
	fileList, err := filepath.Glob(testFile)
	if err != nil {
		t.Errorf("Unexpected error getting list of files in directory: %v", err)
	}

	// Check if any files are left that match the test file
	if len(fileList) > 0 {
		t.Errorf("Not all metric log files were deleted:\n\t%v", fileList)
	}

	// Remove non-matching files
	_ = os.RemoveAll(filepath.Dir(testFile))
}

// Creates a specified number of files at the specified path for testing.
func createTempFiles(path string, num int) error {
	for i := 0; i < num; i++ {
		// Convert the index to a string
		indexString := strconv.FormatUint(uint64(i), 10)

		// Replace the symbol placeholder with the index
		file := strings.ReplaceAll(path, logFilePlaceholder, indexString)

		// Write the file
		err := utils.WriteFile(file, []byte("test"), utils.FilePerms, utils.DirPerms)
		if err != nil {
			return err
		}
	}

	// Write some more files that do not conform to the path pattern
	err := utils.WriteFile(filepath.Join(filepath.Dir(path), "testA.txt"),
		[]byte("test"), utils.FilePerms, utils.DirPerms)
	if err != nil {
		return err
	}

	err = utils.WriteFile(filepath.Join(filepath.Dir(path), "testB.txt"),
		[]byte("test"), utils.FilePerms, utils.DirPerms)
	if err != nil {
		return err
	}

	return nil
}

func TestGatherMetrics(t *testing.T) {

}
