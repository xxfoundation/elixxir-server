///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package node

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"git.xx.network/elixxir/server/internal"
	"git.xx.network/elixxir/server/internal/measure"
	"git.xx.network/elixxir/server/io"
	"git.xx.network/xx_network/primitives/id"
	"git.xx.network/xx_network/primitives/utils"
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
)

// GatherMetrics retrieves the roundMetrics for each node, converts it to JSON,
// and writes them to a log file.
func GatherMetrics(instance *internal.Instance, roundID id.Round) error {
	// Get metrics for all nodes
	rm := instance.GetRoundManager()
	r, err := rm.GetRound(roundID)
	if err != nil {
		return errors.WithMessagef(err, "Failed to get round with id %+v", roundID)
	}
	roundMetrics, err := io.TransmitGetMeasure(instance.GetNetwork(),
		r.GetTopology(), roundID)

	// Convert the roundMetrics array to JSON
	jsonData, err := buildMetricJSON(roundMetrics, false)
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

// ClearMetricsLogs deletes all metric logs matching the specified path. For
// matching the correct files, the logFilePlaceholder must be an asterisk
// character (*).
//
// This function is intended to be run at server startup to clear out the metric
// log files from the previous server instance. It is assumed that the metrics
// log path is unchanged from the previous server run.
func ClearMetricsLogs(path string) error {
	// Expand and clean the path
	path, err := utils.ExpandPath(path)
	if err != nil {
		return err
	}

	// Get a list of all files matching the specified path
	fileList, err := filepath.Glob(path)
	if err != nil {
		return err
	}

	// Loop through all the matching files
	for _, file := range fileList {
		// Remove the log file
		err = os.Remove(file)

		if err != nil {
			return err
		}
	}

	return nil
}
