package services

type Stream interface {
	GetStreamName() string
	Link(BatchSize uint32, source ...interface{})
}
