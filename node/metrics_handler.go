package node

import (
	"encoding/json"
	"errors"
	"fmt"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/primitives/utils"
	"gitlab.com/elixxir/server/io"
	"gitlab.com/elixxir/server/server"
	"gitlab.com/elixxir/server/server/measure"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	// Symbol placeholder that specifies where to put a unique identifier in the
	// log file name
	logFilePlaceholder = "*"

	// Character(s) to be printed at the start of each indented JSON line
	jsonPrefix = ""

	// Character(s) to be printed as the indent for each indented JSON line
	jsonIndent = "\t"

	// Separator for multiple error messages
	errorDelimiter = "; "
)

// GatherMetrics retrieves the roundMetrics for each node, converts it to JSON,
// and writes them to a log file.
func GatherMetrics(instance *server.Instance, roundID id.Round, whitespace bool) error {
	// Get metrics for all nodes
	roundMetrics, err := io.TransmitGetMeasure(instance.GetNetwork(),
		instance.GetTopology(), roundID)

	// Convert the roundMetrics array to JSON
	jsonData, err := buildMetricJSON(roundMetrics, whitespace)
	if err != nil {
		return err
	}

	// Save JSON to log file
	err = saveMetricJSON(jsonData, instance.GetMetricsLog(), roundID)

	return err
}

// buildMetricJSON converts the roundMetrics array to a JSON string. If the
// whitespace flag is set, then each new JSON element will appear on its own
// line with an indent.
func buildMetricJSON(roundMetrics []measure.RoundMetrics, whitespace bool) ([]byte, error) {
	var data []byte
	var err error

	// If the whitespace flag is set, then the metrics JSON will be indented
	if whitespace {
		data, err = json.MarshalIndent(roundMetrics, jsonPrefix, jsonIndent)
	} else {
		data, err = json.Marshal(roundMetrics)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to JSON marshal round metrics: %v", err)
	}

	return data, err
}

// saveMetricJSON writes the metric JSON data to the specified log file. The
// placeholder in the log file name is replaced with the round ID.
func saveMetricJSON(jsonData []byte, logFileName string, roundID id.Round) error {
	// Convert round ID to a string
	roundIDString := strconv.FormatUint(uint64(roundID), 10)

	// Replace the symbol placeholder with the round ID
	path := strings.ReplaceAll(logFileName, logFilePlaceholder, roundIDString)

	// Write the JSON data to the specified file
	err := utils.WriteFile(path, jsonData, utils.FilePerms, utils.DirPerms)

	if err != nil {
		return fmt.Errorf("failed to write metrics log file %s: %v", path, err)
	}

	return err
}

// ClearMetricsLogs deletes all metric logs matching the specified path.
func ClearMetricsLogs(path string) error {
	var errs []string

	// Expand and clean the path
	path, err := utils.ExpandPath(path)
	if err != nil {
		return err
	}

	// Get a list of all the files in the directory
	fileList, err := ioutil.ReadDir(filepath.Dir(path))
	if err != nil {
		return err
	}

	for i := range fileList {
		// Convert the index to a string
		indexString := strconv.FormatUint(uint64(i), 10)

		// Replace the symbol placeholder with the index
		logFile := strings.ReplaceAll(path, logFilePlaceholder, indexString)

		// Remove the log file
		err = os.Remove(logFile)

		if err != nil {
			errs = append(errs, err.Error())
		}
	}

	// If errors occurred above, then concatenate them into a new error to be
	// returned
	var errReturn error
	if len(errs) > 0 {
		errReturn = errors.New(strings.Join(errs, errorDelimiter))
	}

	return errReturn
}
