package access_log

type NullAccessLogger struct {
}

func (x *NullAccessLogger) Run()                {}
func (x *NullAccessLogger) Stop()               {}
func (x *NullAccessLogger) Log(AccessLogRecord) {}
