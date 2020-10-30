///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package permissioning

// Saves a list of node IP addresses in the NDF to a file. A separate file
// contains the line index of this node's IP address in the IP list file.

import (
	"bytes"
	"github.com/pkg/errors"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/ndf"
	"gitlab.com/xx_network/primitives/utils"
	"net"
	"strconv"
	"strings"
)

// The amount to increment each port.
const portIncrementAmount = 128

// The string added to the end of the filename for the file that contains the
// index of this node's IP in the IP list file.
const indexFileNameSuffix = "-index"

// The value printed in the index file if the node's IP does not appear in the
// list.
const unknownIndexValue = -1

// SaveNodeIpList saves a list of all node addresses in the NDF to a newline
// delineated file in the same order they appear in the NDF. The port of each
// address is modified.
func SaveNodeIpList(n *ndf.NetworkDefinition, path string, nid *id.ID) error {
	// Convert NDF to newline delineated list of node addresses
	data, index, err := getListOfNodeIPs(n, nid)
	if err != nil {
		return err
	}

	// Write list to file path
	err = utils.WriteFile(path, []byte(data), utils.FilePerms, utils.DirPerms)
	if err != nil {
		return errors.Errorf("Failed to save IP list file: %v", err)
	}

	// Write index to file path
	indexPath := modifyPath(path, indexFileNameSuffix)
	err = utils.WriteFile(indexPath, []byte(strconv.FormatInt(int64(index), 10)), utils.FilePerms, utils.DirPerms)
	if err != nil {
		return errors.Errorf("Failed to save IP index file: %v", err)
	}

	return nil
}

// getListOfNodeIPs returns a newline (\n) delineated list of node IP addresses
// listed in the the NDF in the same order. It also returns the line index of
// this node's IP. Each address's port is incremented by a set amount. An error
// is returned if the NDF cannot be decoded or if the address cannot be parsed
// correctly.
func getListOfNodeIPs(n *ndf.NetworkDefinition, nid *id.ID) (string, int, error) {
	index := unknownIndexValue

	// Generate array of addresses
	ipList := make([]string, len(n.Nodes))
	for i, n := range n.Nodes {
		u, err := incrementPort(n.Address)
		if err != nil {
			return "", unknownIndexValue, err
		} else {
			ipList[i] = u
		}

		// Save the index if the node IDs match
		if index == unknownIndexValue && bytes.Equal(nid.Marshal(), n.ID) {
			index = i
		}
	}

	// Combine array into newline separated list
	list := strings.Join(ipList, "\n")

	// Join addresses into newline delineated string
	return list, index, nil
}

// incrementPort increments the address's port by a set amount. If the resulting
// port would be greater than 65535, then the port is decremented instead. The
// modified address with the modified port is returned.
//
// An error is returned if the address is not of the form `host:port`, or if the
// port is not an integer.
func incrementPort(address string) (string, error) {
	// Separate host and port
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return "", errors.Errorf("Failed to parse IP address: %v", err)
	}

	// Convert port to integer
	portNum, err := strconv.Atoi(port)
	if err != nil {
		return "", errors.Errorf("Failed to parse IP address port: %v", err)
	}

	// Increment the port
	portNum += portIncrementAmount

	// If the port is larger than the max port, then decrement the port instead
	if portNum > 65535 {
		portNum = portNum - (2 * portIncrementAmount)
	}

	// Combine host and port and return as single address
	return net.JoinHostPort(host, strconv.Itoa(portNum)), nil
}

// modifyPath adds a suffix to the path's filename (after the extension).
func modifyPath(path, suffix string) string {
	return path + suffix
}
