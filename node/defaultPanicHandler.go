package node

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/internal"
)

func GetDefaultPanicHanlder(i *internal.Instance, roundID id.Round) func(g, m string, err error) {
	return func(g, m string, err error) {
		roundErr := errors.Errorf("Error in module %s of graph %s: %+v", g,
			m, err)
		i.ReportRoundFailure(roundErr, i.GetID(), roundID)
	}
}
