package access_log

type AccessLogger interface {
	Run()
	Stop()
	Log(record AccessLogRecord)
}
