package common

import (
	slog "log"
	"fmt"
	"time"
	"os"
	"path/filepath"
)

var logfile *os.File

func init(){
	var err error
	logPath := filepath.Join(GetCurrentDir(), LOG_PATH)
	os.MkdirAll(logPath, os.ModePerm)
	logfile, err = os.OpenFile(filepath.Join(logPath, "access.log"), os.O_WRONLY|os.O_CREATE, os.ModePerm)
	if err != nil {
		slog.Printf("ERROR log file: %+v", err)
	}
	logfile.WriteString("")
}

type LogEntry struct {
	Host       string    `json:"host"`
	Method     string    `json:"method"`
	RequestURI string    `json:"pathname"`
	Proto      string    `json:"proto"`
	Status     int       `json:"status"`
	Scheme     string    `json:"scheme"`
	UserAgent  string    `json:"userAgent"`
	Ip         string    `json:"ip"`
	Referer    string    `json:"referer"`
	Timestamp  time.Time `json:"_id"`
	Duration   int64     `json:"responseTime"`
	Version    string    `json:"version"`
	Backend    string    `json:"backend"`
}

type log struct{
	enable bool

	debug  bool
	info   bool
	warn   bool
	error  bool
}

func (l *log) Info(format string, v ...interface{}) {
	if l.info && l.enable {
		message := fmt.Sprintf("INFO  " + format + "\n", v...)
		logfile.WriteString(message)
	}
}

func (l *log) Warning(format string, v ...interface{}) {
	if l.warn && l.enable {
		message := fmt.Sprintf("WARN  " + format + "\n", v...)
		logfile.WriteString(message)
	}
}

func (l *log) Error(format string, v ...interface{}) {
	if l.error && l.enable {
		message := fmt.Sprintf("ERROR " + format + "\n", v...)
		logfile.WriteString(message)
	}
}

func (l *log) Debug(format string, v ...interface{}) {
	if l.debug && l.enable {
		message := fmt.Sprintf("DEBUG " + format + "\n", v...)
		logfile.WriteString(message)
	}
}

func (l *log) Stdout(format string, v ...interface{}) {
	slog.Printf(format + "\n", v...)
}

func (l *log) Close() {
	logfile.Close()
}

func (l *log) SetVisibility(str string) {
	switch str {
	case "WARNING":
		l.debug = false
		l.info = false
		l.warn = true
		l.error = true
	case "ERROR":
		l.debug = false
		l.info = false
		l.warn = false
		l.error = true
	case "DEBUG":
		l.debug = true
		l.info = true
		l.warn = true
		l.error = true
	case "INFO":
		l.debug = false
		l.info = true
		l.warn = true
		l.error = true
	default:
		l.debug = false
		l.info = true
		l.warn = true
		l.error = true
	}
}

func(l *log) Enable(val bool) {
	l.enable = val
}

var Log = func () log {
	l := log{}
	l.Enable(true)
	return l
}()
