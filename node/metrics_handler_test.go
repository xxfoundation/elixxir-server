package node

import (
	"bytes"
	"encoding/json"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/primitives/utils"
	"gitlab.com/elixxir/server/server"
	"gitlab.com/elixxir/server/server/measure"
	"os"
	"reflect"
	"strconv"
	"testing"
)

// This will error when the type does not match the function
var _ server.MetricsHandler = GatherMetrics

// Tests  that buildMetricJSON marshals data correctly by unmarshalling the JSON
// it outputs and comparing it to the original structure.
func Test_buildMetricJSON(t *testing.T) {
	roundMetrics := []measure.RoundMetrics{
		measure.NewRoundMetrics(id.Round(10), 5),
		measure.NewRoundMetrics(id.Round(20), 15),
		measure.NewRoundMetrics(id.Round(30), 25),
	}

	var rmTest []measure.RoundMetrics

	data, err := buildMetricJSON(roundMetrics)

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
