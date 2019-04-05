package graphs

import "gitlab.com/elixxir/server/services"

type Initializer func(batchSize uint32, errorHandler services.ErrorCallback) *services.Graph
