package main

import (
	"fmt"
	"os"
	"sync"
	"time"
)

const (
	LOG_LEVEL_DEBUG = 0
	LOG_LEVEL_INFO  = 1
	LOG_LEVEL_WARN  = 2
	LOG_LEVEL_ERROR = 3
	LOG_LEVEL_FATAL = 4
)

type Logger interface {
	Info(string, ...interface{})
	Error(string, ...interface{})
	Debug(string, ...interface{})
	Warn(string, ...interface{})
	Fatal(string, ...interface{})
	Infof(string, ...interface{})
	Errorf(string, ...interface{})
	Debugf(string, ...interface{})
	Warnf(string, ...interface{})
	Fatalf(string, ...interface{})
}

type MultiPingLogger struct {
	sync.Mutex
	out   *os.File
	Level int
}

var Log *MultiPingLogger

func SetLogOutput(logger *os.File) {
	Log.out = logger
}

func SetLogLevel(level int) {
	Log.Level = level
}

func init() {
	Log = &MultiPingLogger{
		out:   os.Stdout,
		Level: LOG_LEVEL_INFO,
	}

	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel != "" {
		switch logLevel {
		case "DEBUG":
			Log.Level = LOG_LEVEL_DEBUG
		case "INFO":
			Log.Level = LOG_LEVEL_INFO
		case "WARN":
			Log.Level = LOG_LEVEL_WARN
		case "ERROR":
			Log.Level = LOG_LEVEL_ERROR
		case "FATAL":
			Log.Level = LOG_LEVEL_FATAL
		default:
			Log.Level = LOG_LEVEL_INFO
		}
	}
}

func (l *MultiPingLogger) Info(value string, args ...interface{}) {

	if l.Level > LOG_LEVEL_INFO {
		return
	}
	l.Lock()
	defer l.Unlock()

	t := time.Now().Format("2006-01-02 15:04:05")
	l.out.WriteString(t + " INFO: " + value)
	for _, arg := range args {
		l.out.WriteString(fmt.Sprintf("%v", arg))
	}

	l.out.WriteString("\n")
}

func (l *MultiPingLogger) Debug(value string, args ...interface{}) {

	if l.Level > LOG_LEVEL_DEBUG {
		return
	}

	l.Lock()
	defer l.Unlock()
	t := time.Now().Format("2006-01-02 15:04:05")
	l.out.WriteString(t + " DEBUG: " + value)
	for _, arg := range args {
		l.out.WriteString(fmt.Sprintf("%v", arg))
	}

	l.out.WriteString("\n")
}

func (l *MultiPingLogger) Error(value string, args ...interface{}) {

	if l.Level > LOG_LEVEL_ERROR {
		return
	}
	l.Lock()
	defer l.Unlock()

	t := time.Now().Format("2006-01-02 15:04:05")
	l.out.WriteString(t + " ERROR: " + value)
	for _, arg := range args {
		l.out.WriteString(fmt.Sprintf("%v", arg))
	}

	l.out.WriteString("\n")
}

func (l *MultiPingLogger) Warn(value string, args ...interface{}) {

	if l.Level > LOG_LEVEL_WARN {
		return
	}
	l.Lock()
	defer l.Unlock()
	t := time.Now().Format("2006-01-02 15:04:05")
	l.out.WriteString(t + " WARN: " + value)
	for _, arg := range args {
		l.out.WriteString(fmt.Sprintf("%v", arg))
	}

	l.out.WriteString("\n")
}

func (l *MultiPingLogger) Fatal(value string, args ...interface{}) {

	if l.Level > LOG_LEVEL_FATAL {
		return
	}
	l.Lock()
	defer l.Unlock()

	t := time.Now().Format("2006-01-02 15:04:05")
	l.out.WriteString(t + " FATAL: " + value)
	for _, arg := range args {
		l.out.WriteString(fmt.Sprintf("%v", arg))
	}

	l.out.WriteString("\n")
	os.Exit(1)
}

/*
 * ----------------------- Printf functions ------------------------------
 */
func (l *MultiPingLogger) Infof(value string, args ...interface{}) {

	if l.Level > LOG_LEVEL_INFO {
		return
	}
	l.Lock()
	defer l.Unlock()

	t := time.Now().Format("2006-01-02 15:04:05")
	l.out.WriteString(t + " INFO: " + fmt.Sprintf(value, args...))
	l.out.WriteString("\n")
}

func (l *MultiPingLogger) Debugf(value string, args ...interface{}) {

	if l.Level > LOG_LEVEL_DEBUG {
		return
	}
	l.Lock()
	defer l.Unlock()

	t := time.Now().Format("2006-01-02 15:04:05")
	l.out.WriteString(t + " DEBUG: " + fmt.Sprintf(value, args...))
	l.out.WriteString("\n")
}

func (l *MultiPingLogger) Errorf(value string, args ...interface{}) {

	if l.Level > LOG_LEVEL_ERROR {
		return
	}

	l.Lock()
	defer l.Unlock()
	t := time.Now().Format("2006-01-02 15:04:05")
	l.out.WriteString(t + " ERROR: " + fmt.Sprintf(value, args...))

	l.out.WriteString("\n")
}

func (l *MultiPingLogger) Warnf(value string, args ...interface{}) {

	if l.Level > LOG_LEVEL_WARN {
		return
	}

	l.Lock()
	defer l.Unlock()
	t := time.Now().Format("2006-01-02 15:04:05")
	l.out.WriteString(t + " WARN: " + fmt.Sprintf(value, args...))

	l.out.WriteString("\n")
}

func (l *MultiPingLogger) Fatalf(value string, args ...interface{}) {

	if l.Level > LOG_LEVEL_FATAL {
		return
	}
	l.Lock()
	defer l.Unlock()
	t := time.Now().Format("2006-01-02 15:04:05")
	l.out.WriteString(t + " FATAL: " + fmt.Sprintf(value, args...))

	l.out.WriteString("\n")
	os.Exit(1)
}
