////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package conf

var ExpectedPaths = Paths{
	Idf:          "nodeID.json",
	Cert:         "~/.elixxir/cert.crt",
	Key:          "~/.elixxir/key.pem",
	Log:          "~/.elixxir/server.log",
	ipListOutput: "/opt/xxnetwork/node-logs/ipList.txt",
}
