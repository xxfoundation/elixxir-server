package services

type Stream interface {
	GetName() string
	Link(BatchSize uint32, source ...interface{})
}
