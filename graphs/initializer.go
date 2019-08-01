////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package graphs

import "gitlab.com/elixxir/server/services"

// Initializer is the function type signature for how all graphs should be initialized
type Initializer func(gc services.GraphGenerator) *services.Graph
