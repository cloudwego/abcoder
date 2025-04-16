/**
 * Copyright 2025 ByteDance Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package utils

import (
	"fmt"
	"runtime"
)

type withMessage struct {
	file  string
	line  int
	cause error
	msg   string
}

func (e *withMessage) Error() string {
	return fmt.Sprintf("%s:%d: %s\n%v", e.file, e.line, e.msg, e.cause)
}

func WrapError(err error, msg string, v ...interface{}) error {
	_, file, line, ok := runtime.Caller(1)
	if !ok {
		file = "???"
		line = 0
	}
	if len(v) > 0 {
		msg = fmt.Sprintf(msg, v...)
	}
	return &withMessage{
		file:  file,
		line:  line,
		cause: err,
		msg:   msg,
	}
}
