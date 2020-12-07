package io

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/id"
	"strconv"
	"strings"
)

type checkPermConnComm interface {
	GetHost(hostId *id.ID) (*connect.Host, bool)
	SendCheckConnectivityMessage(host *connect.Host, message *pb.Address) (*pb.ConnectivityResponse, error)
}

func TransmitSendCheckConnectivity(addr string, port int, comms checkPermConnComm) (string, error) {
	// Get the permissioning host so we can check our connection to it
	permHost, found := comms.GetHost(&id.Permissioning)
	if found == false {
		return "", errors.New("CheckPermConn could not find permissioning host")
	}

	// Build our Address object to send in the message
	addrMsg := pb.Address{IP: "", Port: strconv.Itoa(port)}
	if addr != "0.0.0.0" {
		addrMsg.IP = addr
	}

	// Send check connectivity message
	r, err := comms.SendCheckConnectivityMessage(permHost, &addrMsg)
	if err != nil {
		return "", err
	}

	if addr != "0.0.0.0" {
		// Print out information about the test to the user
		if strings.Compare(r.CallerAddr, addr) != 0 {
			jww.INFO.Printf("Address used in config does not match address detected from CheckConnectivity test." +
				"This likely means your Node is sending data out via an IP other than the one in your config file!")
		}
		return addr, nil
	} else {
		if strings.Contains(r.CallerAddr, ":") {
			r.CallerAddr = strings.Split(r.CallerAddr, ":")[0]
		}
		jww.INFO.Printf("Detected IP/port is available: %t\n The IP permissioning detected"+
			" from your request to it is \"%s\"\n", r.GetCallerAvailable(), r.CallerAddr)
		jww.INFO.Printf("IP in config is available: %t\n", r.GetOtherAvailable())
		return r.CallerAddr, nil
	}
}
