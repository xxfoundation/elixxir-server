package node

import (
	"encoding/json"
	"fmt"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/comms/utils"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/io"
	"gitlab.com/elixxir/server/server"
	"gitlab.com/elixxir/server/server/measure"
	"io/ioutil"
	"strconv"
	"strings"
)

const (
	// Symbol placeholder that specifies where to put a unique identifier in the
	// log file name
	logFilePlaceholder = "*"

	// Log file permissions in octal; user: read/write, group: read, other: read
	logFilePermissions = 0644
)

// GatherMetrics retrieves the roundMetrics for each node, converts it to JSON,
// and writes them to a log file.
func GatherMetrics(instance *server.Instance, roundID id.Round) error {
	jww.INFO.Printf("Gathering metrics data for round %d.", roundID)
	roundMetrics, err := io.TransmitGetMeasure(instance.GetNetwork(),
		instance.GetTopology(), roundID)

	// Convert the roundMetrics array to JSON
	jww.INFO.Printf("Building metrics JSON for round %d.", roundID)
	jsonData, err := buildMetricJSON(roundMetrics)
	if err != nil {
		return err
	}

	// Save JSON to log file
	jww.INFO.Printf("Saving metrics JSON for round %d.", roundID)
	err = saveMetricJSON(jsonData, instance.GetMetricsLog(), roundID)

	return err
}

// buildMetricJSON converts the roundMetrics array to a JSON string.
func buildMetricJSON(roundMetrics []measure.RoundMetrics) ([]byte, error) {
	data, err := json.MarshalIndent(roundMetrics, "", "\t")

	if err != nil {
		return nil, fmt.Errorf("failed to JSON marshal round metrics: %v", err)
	}

	return data, err
}

// saveMetricJSON writes the metric JSON data to the specified log file. The
// placeholder in the log file name is replaced with the round ID.
func saveMetricJSON(jsonData []byte, logFileName string, roundID id.Round) error {
	// Replace the symbol placeholder with the round ID
	path := strings.ReplaceAll(logFileName, logFilePlaceholder,
		strconv.FormatUint(uint64(roundID), 10))

	// Get the full file path by resolving the "~" character
	path = utils.GetFullPath(path)

	// Write the JSON data to the specified file
	err := ioutil.WriteFile(path, jsonData, logFilePermissions)

	if err != nil {
		return fmt.Errorf("failed to write metrics log file %s: %v", path, err)
	}

	return err
}
