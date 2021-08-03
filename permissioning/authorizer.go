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

type Sender struct {
	Send func(host *connect.Host) (interface{}, error)
	Name string
}

func (s Sender) String() string {
	return s.Name
}

const AuthorizationFailure = "failed to authorize"

// Authorize will send an authorization request with the authorizer server.
func Authorize(instance *internal.Instance, authHost *connect.Host) error {

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
		TimeStamp: authorizerTimestamp.UnixNano(),
		Signature: authorizerSig,
	}

	// Send authorization request
	_, err = instance.GetNetwork().SendAuthorizerAuth(authHost, authorizerMsg)
	if err != nil {
		return errors.Errorf("%s: %v", AuthorizationFailure, err)
	}

	return nil
}

// Send will attempt to send a message to permissioning. If the node cannot connect,
// it will attempt to authorize itself with the authorizer. If successful, it will
// try to send the message again
func Send(sender Sender, instance *internal.Instance, authHost *connect.Host) (response interface{}, err error) {

	// Fetch the host information from the network
	permHost, ok := instance.GetNetwork().GetHost(&id.Permissioning)
	if !ok {
		return nil, errors.New("Could not get permissioning host")
	}
	response, err = sender.Send(permHost)
	if authHost == nil || (err != nil &&
		!strings.Contains(strings.ToLower(err.Error()), "giving up")) {
		return response, err
	}

	// Attempt to authorize
	jww.WARN.Printf("Failed to send %s to permissioning "+
		"due to potential authorization error, attempting to authorize...", sender.String())

	for err = Authorize(instance, authHost); err != nil; {
	}

	// If we had to authorize, retry the comm again
	// now that authorization was successful
	jww.WARN.Printf("Resending %s after successful authorization", sender.String())

	return sender.Send(permHost)
}
