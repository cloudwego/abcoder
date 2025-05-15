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

type LogLevel uint8

const (
	ErrorLevel LogLevel = 1
	InfoLevel  LogLevel = 2
	DebugLevel LogLevel = 3
)

var (
	errlogger   = log.New(os.Stderr, "[ERROR]", log.Ltime|log.Lshortfile)
	infologger  = log.New(os.Stderr, "[INFO]", log.Ltime|log.Lshortfile)
	debuglogger = log.New(os.Stderr, "[DEBUG]", log.Ltime|log.Lshortfile)
)

var logLevel LogLevel = ErrorLevel

func SetLogLevel(level LogLevel) {
	logLevel = level
}

func Info(s string, args ...interface{}) {
	if logLevel < InfoLevel {
		return
	}
	s = fmt.Sprintf(s, args...)
	infologger.Output(2, s)
}

func Info_skip(calldepth int, s string, args ...interface{}) {
	if logLevel < InfoLevel {
		return
	}
	s = fmt.Sprintf(s, args...)
	infologger.Output(2+calldepth, s)
}

func Debug(s string, args ...interface{}) {
	if logLevel < DebugLevel {
		return
	}
	s = fmt.Sprintf(s, args...)
	debuglogger.Output(2, s)
}

func Error(s string, args ...interface{}) {
	if logLevel < ErrorLevel {
		return
	}
	s = fmt.Sprintf(s, args...)
	errlogger.Output(2, s)
}

func Error_skip(calldepth int, s string, args ...interface{}) {
	if logLevel < ErrorLevel {
		return
	}
	s = fmt.Sprintf(s, args...)
	errlogger.Output(2+calldepth, s)
}
