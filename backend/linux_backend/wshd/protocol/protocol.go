package protocol

type RequestMessage struct {
	User string
	Argv []string
}

type ResponseMessage struct{}

type ExitStatusMessage struct {
	ExitStatus int
}
