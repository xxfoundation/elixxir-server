package round

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/comms/mixmessages"
)

type Queue chan *mixmessages.RoundInfo

func NewQueue()Queue{
	return make(chan *mixmessages.RoundInfo, 1)
}

func (rq Queue)Send(ri *mixmessages.RoundInfo)error{
	select{
	case rq<-ri:
		return nil
	default:
		return errors.New("Round Queue is full")
	}
}

func (rq Queue)Receive()(*mixmessages.RoundInfo, error){
	select{
	case ri:=<-rq:
		return ri, nil
	default:
		return nil, errors.New("Round Queue is empty")
	}
}
