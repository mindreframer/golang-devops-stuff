package errors

type SentinelCommandError struct {
	Error      error
	ErrorClass string
	Command    string
}

type SentinelConnectionError struct {
	SentinelName    string
	SentinelAddress string
	SentinelPort    string
	Error           error
	ErrorClass      string
}
