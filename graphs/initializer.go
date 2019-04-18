package graphs

import "gitlab.com/elixxir/server/services"

type Initializer func(gc services.GraphGenerator) *services.Graph
