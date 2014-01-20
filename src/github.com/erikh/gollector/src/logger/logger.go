package logger

import (
	"fmt"
	"log/syslog"
)

var facilityMap = map[string]syslog.Priority{
	"kern":     syslog.LOG_KERN,
	"user":     syslog.LOG_USER,
	"mail":     syslog.LOG_MAIL,
	"daemon":   syslog.LOG_DAEMON,
	"auth":     syslog.LOG_AUTH,
	"syslog":   syslog.LOG_SYSLOG,
	"lpr":      syslog.LOG_LPR,
	"news":     syslog.LOG_NEWS,
	"uucp":     syslog.LOG_UUCP,
	"cron":     syslog.LOG_CRON,
	"authpriv": syslog.LOG_AUTHPRIV,
	"ftp":      syslog.LOG_FTP,
	"local0":   syslog.LOG_LOCAL0,
	"local1":   syslog.LOG_LOCAL1,
	"local2":   syslog.LOG_LOCAL2,
	"local3":   syslog.LOG_LOCAL3,
	"local4":   syslog.LOG_LOCAL4,
	"local5":   syslog.LOG_LOCAL5,
	"local6":   syslog.LOG_LOCAL6,
	"local7":   syslog.LOG_LOCAL7,
}

var priorityMap = map[string]syslog.Priority{
	"emerg":   syslog.LOG_EMERG,
	"alert":   syslog.LOG_ALERT,
	"crit":    syslog.LOG_CRIT,
	"err":     syslog.LOG_ERR,
	"warning": syslog.LOG_WARNING,
	"notice":  syslog.LOG_NOTICE,
	"info":    syslog.LOG_INFO,
	"debug":   syslog.LOG_DEBUG,
}

var priorityCallMap = map[string]func(*syslog.Writer, string) error{
	"emerg":   (*syslog.Writer).Emerg,
	"alert":   (*syslog.Writer).Alert,
	"crit":    (*syslog.Writer).Crit,
	"err":     (*syslog.Writer).Err,
	"warning": (*syslog.Writer).Warning,
	"notice":  (*syslog.Writer).Notice,
	"info":    (*syslog.Writer).Info,
	"debug":   (*syslog.Writer).Debug,
}

type Logger struct {
	LogLevel string
	Facility string
	writer   *syslog.Writer
}

func Init(facility string, priority string) *Logger {
	var err error
	log := &Logger{
		LogLevel: priority,
		Facility: facility,
	}

	log.writer, err = syslog.New(facilityMap[facility]|priorityMap[priority], "gollector")

	if err != nil {
		panic(fmt.Sprintf("Cannot connect to syslog: %s", err))
	}

	err = log.Log("info", "Initialized Logger")
	if err != nil {
		panic(fmt.Sprintf("Cannot write to syslog: %s", err))
	}

	return log
}

func (log *Logger) Log(priority string, m string) error {
	if priorityMap[priority] <= priorityMap[log.LogLevel] {
		return priorityCallMap[priority](log.writer, m)
	}

	return nil
}

func (log *Logger) Close() {
	log.writer.Close()
}
