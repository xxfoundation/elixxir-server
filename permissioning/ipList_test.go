///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package permissioning

import (
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/ndf"
	"gitlab.com/xx_network/primitives/utils"
	"os"
	"strconv"
	"testing"
)

// Tests happy path of SaveNodeIpList().
func TestSaveNodeIpList(t *testing.T) {
	path := "testIpList.txt"
	nodes, expectedList := makeTestNodes(t)

	// Delete the list file at the end
	defer func() {
		err := os.RemoveAll(path)
		if err != nil {
			t.Errorf("Error deleting test file %#v:\n%v", path, err)
		}
		err = os.RemoveAll(modifyPath(path, indexFileNameSuffix))
		if err != nil {
			t.Errorf("Error deleting test file %#v:\n%v", modifyPath(path, indexFileNameSuffix), err)
		}
	}()

	err := SaveNodeIpList(&ndf.NetworkDefinition{Nodes: nodes}, path, id.NewIdFromUInt(2, id.Node, t))
	if err != nil {
		t.Errorf("SaveNodeIpList() produced an error: %v", err)
	}

	data, err := utils.ReadFile(path)
	if err != nil {
		t.Errorf("Failed to read file: %v", err)
	}

	if string(data) != expectedList {
		t.Errorf("SaveNodeIpList() did not save the correct contents."+
			"\n\texpected: %#v\n\treceived: %v", expectedList, string(data))
	}

	data, err = utils.ReadFile(modifyPath(path, indexFileNameSuffix))
	if err != nil {
		t.Errorf("Failed to read file: %v", err)
	}

	if string(data) != "2" {
		t.Errorf("SaveNodeIpList() did not save the correct contents."+
			"\n\texpected: %#v\n\treceived: %#v", "2", string(data))
	}
}

// Tests that SaveNodeIpList() produces an error when the provided path is bad.
func TestSaveNodeIpList_FileError(t *testing.T) {
	path := "~a/testIpList.txt"
	nodes, _ := makeTestNodes(t)

	// Delete the list file at the end
	defer func() {
		err := os.RemoveAll(path)
		if err != nil {
			t.Errorf("Error deleting test file %#v:\n%v", path, err)
		}
		err = os.RemoveAll(modifyPath(path, indexFileNameSuffix))
		if err != nil {
			t.Errorf("Error deleting test file %#v:\n%v", modifyPath(path, indexFileNameSuffix), err)
		}
	}()

	err := SaveNodeIpList(&ndf.NetworkDefinition{Nodes: nodes}, path, id.NewIdFromUInt(2, id.Node, t))
	if err == nil {
		t.Errorf("SaveNodeIpList() did not produce an error.")
	}

	_, err = utils.ReadFile(path)
	if err == nil {
		t.Error("SaveNodeIpList() created a file when it should not have.")
	}
	_, err = utils.ReadFile(modifyPath(path, indexFileNameSuffix))
	if err == nil {
		t.Error("SaveNodeIpList() created a file when it should not have.")
	}
}

// Tests that happy path of getListOfNodeIPs().
func Test_getListOfNodeIPs(t *testing.T) {
	nodes, expectedList := makeTestNodes(t)

	list, index, err := getListOfNodeIPs(&ndf.NetworkDefinition{Nodes: nodes}, id.NewIdFromUInt(2, id.Node, t))

	if err != nil {
		t.Errorf("getListOfIPs() produced an error: %v", err)
	}

	if expectedList != list {
		t.Errorf("getListOfIPs() produced incorrect list."+
			"\n\texpected: %#v\n\treceived: %#v", expectedList, list)
	}

	if index != 2 {
		t.Errorf("getListOfIPs() produced incorrect index."+
			"\n\texpected: %d\n\treceived: %d", 2, index)
	}
}

// Tests that happy path of getListOfNodeIPs().
func Test_getListOfNodeIPs_Error(t *testing.T) {
	nodes, _ := makeTestNodes(t)

	nodes = append(nodes, ndf.Node{ID: []byte{5, 23, 2}, Address: "192.168.1.1"})

	list, index, err := getListOfNodeIPs(&ndf.NetworkDefinition{Nodes: nodes}, id.NewIdFromUInt(2, id.Node, t))

	if err == nil {
		t.Errorf("getListOfIPs() did not produce an error on invalid IP.")
	}

	if "" != list {
		t.Errorf("getListOfIPs() produced incorrect list."+
			"\n\texpected: %#v\n\treceived: %#v", "", list)
	}

	if index != unknownIndexValue {
		t.Errorf("getListOfIPs() produced incorrect index."+
			"\n\texpected: %d\n\treceived: %d", unknownIndexValue, index)
	}
}

// Tests that happy path of incrementPort().
func Test_incrementPort(t *testing.T) {
	addresses := []string{"0.0.0.0:10000", "192.16.1.172:65534", "[2001:db8:a0b:12f0::1]:21"}
	expectedAddresses := []string{"0.0.0.0:10128", "192.16.1.172:65406", "[2001:db8:a0b:12f0::1]:149"}

	for i, a := range addresses {
		address, err := incrementPort(a)

		if err != nil {
			t.Errorf("incrementPort() produced an error on address %s "+
				"(round %d):\n\t%v", a, i, err)
		}

		if expectedAddresses[i] != address {
			t.Errorf("incrementPort() produced incorrect address (round %d)."+
				"\n\texpected: %s\n\treceived: %s",
				i, expectedAddresses[i], address)
		}
	}
}

// Tests the two error cases of incrementPort().
func Test_incrementPort_Error(t *testing.T) {
	addresses := []string{"0.0.0.0", "192.16.1.172:65hi534", "[2001:db8:a0b:12f0::1]:"}

	for i, a := range addresses {
		address, err := incrementPort(a)

		if err == nil {
			t.Errorf("incrementPort() did not produce an error on address %s "+
				"(round %d).", a, i)
		}

		if "" != address {
			t.Errorf("incrementPort() produced incorrect address (round %d)."+
				"\n\texpected: %s\n\treceived: %s",
				i, "", address)
		}
	}
}

// Tests happy path of modifyPath().
func Test_modifyPath(t *testing.T) {
	suffix := "-suffix"
	paths := []string{"test.txt", "test"}
	expectedPaths := []string{"test.txt" + suffix, "test" + suffix}

	for i, p := range paths {
		testPath := modifyPath(p, suffix)

		if expectedPaths[i] != testPath {
			t.Errorf("modifyPath() returned the wrong path."+
				"\n\texpected: %s\n\treceived: %s", expectedPaths[i], testPath)
		}
	}
}

func makeTestNodes(t *testing.T) ([]ndf.Node, string) {
	ports := []int{11420, 65534, 100, 300}
	expectedPorts := []int{
		ports[0] + portIncrementAmount,
		ports[1] - portIncrementAmount,
		ports[2] + portIncrementAmount,
		ports[3] + portIncrementAmount,
	}
	hosts := []string{"18.237.147.105", "18.237.147.105", "52.11.136.238", "192.16.1.1"}
	nids := []*id.ID{id.NewIdFromUInt(0, id.Node, t), id.NewIdFromUInt(1, id.Node, t), id.NewIdFromUInt(2, id.Node, t), id.NewIdFromUInt(3, id.Node, t)}
	expectedList := ""
	n := make([]ndf.Node, len(ports))

	for i, p := range ports {
		n[i] = ndf.Node{
			ID:             nids[i].Marshal(),
			Address:        hosts[i] + ":" + strconv.Itoa(p),
			TlsCertificate: "-----BEGIN CERTIFICATE-----\nMIIDbDC6m52PyzMNV+2N21IPppKwA==\n-----END CERTIFICATE-----",
		}
		expectedList += hosts[i] + ":" + strconv.Itoa(expectedPorts[i])
		if i < len(ports)-1 {
			expectedList += "\n"
		}
	}

	return n, expectedList
}
