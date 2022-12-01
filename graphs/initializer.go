////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package graphs

import "gitlab.com/elixxir/server/services"

// Initializer is the function type signature for how all graphs should be initialized
type Initializer func(gc services.GraphGenerator) *services.Graph
