package graphs

import "gitlab.com/elixxir/server/services"

type Initializer func(errorHandler services.ErrorCallback) *services.Graph
