package errors

type ServiceError interface {
	GetConstellation()
	GetSentinel()
	GetPod()
	GetError()
	GetErrorClass()
	IsFatal()
}
