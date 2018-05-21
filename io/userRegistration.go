////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package io

import (
	pb "gitlab.com/privategrity/comms/mixmessages"
)

// Check the registration status of a specific user
func (s ServerImpl) PollRegistrationStatus(message *pb.RegistrationPoll) *pb.RegistrationConfirmation {
	return &pb.RegistrationConfirmation{}
}
