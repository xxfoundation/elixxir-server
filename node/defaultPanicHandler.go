package node

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/server/server"
)

func GetDefaultPanicHanlder(i *server.Instance) func(g, m string, err error) {
	return func(g, m string, err error) {
		roundErr := errors.Errorf("Error in module %s of graph %s: %+v", g,
			m, err)
		i.ReportRoundFailure(roundErr, i.GetID())
	}
}
