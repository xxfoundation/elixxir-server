///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package permissioning

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/authorize"
	"gitlab.com/elixxir/server/internal"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/id"
	"strings"
	"time"
)

type SendFunc func(host *connect.Host) (interface{}, error)

const sendRetries = 2

// Authorize will send an authorization request with the authorizer server.
func Authorize(instance *internal.Instance) error {
	// Fetch the host information from the network
	authHost, ok := instance.GetNetwork().GetHost(&id.Authorizer)
	if !ok {
		return errors.Errorf("Could not find host for authorizer")
	}
	// Sign authorization timestamp
	authorizerTimestamp := time.Now()
	authorizerSig, err := authorize.Sign(instance.GetDefinition().RngStreamGen.GetStream(),
		authorizerTimestamp, instance.GetDefinition().PrivateKey)
	if err != nil {
		return errors.Errorf("Unable to sign authorizer timestamp: %v", err)
	}

	// Construct message
	authorizerMsg := &mixmessages.AuthorizerAuth{
		NodeID:    instance.GetID().Bytes(),
		Salt:      instance.GetDefinition().Salt,
		PubkeyPem: instance.GetDefinition().TlsCert,
		TimeStamp: authorizerTimestamp.Unix(),
		Signature: authorizerSig,
	}

	// Send authorization request
	_, err = instance.GetNetwork().SendAuthorizerAuth(authHost, authorizerMsg)
	if err != nil {
		return err
	}

	return nil
}

// Send will attempt to send a message to permissioning. If the node cannot connect,
// it will attempt to authorize itself with the authorizer. If successful, it will
// try to send the message again
func Send(sendFunc SendFunc, instance *internal.Instance) (response interface{}, err error) {

	// Fetch the host information from the network
	permHost, ok := instance.GetNetwork().GetHost(&id.Permissioning)
	if !ok {
		return nil, errors.New("Could not get permissioning host")
	}

	// Attempt to send to permissioning
	for i := 0; i < sendRetries; i++ {
		// Attempt sending message to network
		response, err = sendFunc(permHost)
		if err != nil &&
			(strings.Contains(err.Error(), "connection refused") ||
				strings.Contains(err.Error(), "context deadline exceeded")) {
			jww.WARN.Printf("Could not send to permissioning, "+
				"attempt (%d/%d) to contact authorizer", i+1, sendRetries)

			// If failed, send authorization request
			err = Authorize(instance)
			if err != nil {
				return nil, errors.Errorf("Could not authorize with network: %v", err)
			}
		}

	}

	return response, err
}
