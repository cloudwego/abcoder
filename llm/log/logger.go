// Copyright 2025 CloudWeGo Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package log

import (
	"fmt"
	"log"
	"os"
)

type Logger interface {
	Infof(s string, args ...interface{})
	Errorf(s string, args ...interface{})
	Debugf(s string, args ...interface{})
	Output(calldepth int, s string) error
}

type StdLogger struct {
	errlogger   *log.Logger
	infologger  *log.Logger
	debuglogger *log.Logger
}

func NewStdLogger() *StdLogger {
	return &StdLogger{
		errlogger:   log.New(os.Stderr, "[ERROR]", log.Ltime|log.Lshortfile),
		infologger:  log.New(os.Stderr, "[INFO]", log.Ltime|log.Lshortfile),
		debuglogger: log.New(os.Stderr, "[DEBUG]", log.Ltime|log.Lshortfile),
	}
}

func (l *StdLogger) Infof(s string, args ...interface{}) {
	l.infologger.Output(3, fmt.Sprintf(s, args...))
}

func (l *StdLogger) Errorf(s string, args ...interface{}) {
	l.errlogger.Output(3, fmt.Sprintf(s, args...))
}

func (l *StdLogger) Debugf(s string, args ...interface{}) {
	l.debuglogger.Output(3, fmt.Sprintf(s, args...))
}

func (l *StdLogger) Output(calldepth int, s string) error {
	return l.infologger.Output(calldepth+2, s)
}

type LogLevel uint8

const (
	ErrorLevel LogLevel = 1
	InfoLevel  LogLevel = 2
	DebugLevel LogLevel = 3
)

var logLevel LogLevel = ErrorLevel

func SetLogLevel(level LogLevel) {
	logLevel = level
}

func SetDefaultLogger(logger Logger) {
	defaultLogger = logger
}

var defaultLogger Logger = NewStdLogger()

func Info(s string, args ...interface{}) {
	if logLevel < InfoLevel {
		return
	}
	defaultLogger.Infof(s, args...)
}

func Info_skip(calldepth int, s string, args ...interface{}) {
	if logLevel < InfoLevel {
		return
	}
	s = fmt.Sprintf(s, args...)
	defaultLogger.Output(1+calldepth, s)
}

func Debug(s string, args ...interface{}) {
	if logLevel < DebugLevel {
		return
	}
	defaultLogger.Debugf(s, args...)
}

func Error(s string, args ...interface{}) {
	if logLevel < ErrorLevel {
		return
	}
	defaultLogger.Errorf(s, args...)
}

func Error_skip(calldepth int, s string, args ...interface{}) {
	if logLevel < ErrorLevel {
		return
	}
	s = fmt.Sprintf(s, args...)
	defaultLogger.Output(1+calldepth, s)
}
