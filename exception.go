/*
 * Copyright 2019 the go-netty project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package netty

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// Exception defines an exception
// 异常的接口
type Exception interface {
	// Unwrap inner error.
	Unwrap() error
	// Error message.
	Error() string
	// Stack stack trace.
	Stack() []byte
	// PrintStackTrace dump stack trace to writers.
	PrintStackTrace(writer io.Writer, msg ...string)
}

// AsException to wrap error to Exception
// AsException将错误包装为Exception
func AsException(e interface{}, stack []byte) Exception {

	switch err := e.(type) {
	case nil:
		return nil
	case Exception:
		return err
	case error:
		return exception{error: err, stack: stack}
	default:
		return exception{error: fmt.Errorf("%v", e), stack: stack}
	}
}

// exception impl Exception
// Exception接口实现
type exception struct {
	error error
	stack []byte
}

// Unwrap to unwrap inner error
// exception解析成error
func (e exception) Unwrap() error {
	return e.error
}

// Error to get error message
// 返回error的信息
func (e exception) Error() string {
	return e.error.Error()
}

// Stack to get exception stack trace
// 返回exception中的栈信息
func (e exception) Stack() []byte {
	return e.stack
}

// PrintStackTrace to write stack trance info to writer
// 写入信息
func (e exception) PrintStackTrace(writer io.Writer, msg ...string) {

	// default: write to stderr.
	if nil == writer {
		writer = os.Stderr
	}

	// build output information.
	var sb strings.Builder
	for _, m := range msg {
		sb.WriteString(m)
	}

	sb.WriteString("Error Traceback:\n")
	var err = e.error
	var i int
	for {
		i++
		sb.WriteString(fmt.Sprintf("%T: %s", err, err.Error()))
		if e, ok := err.(interface{ Unwrap() error }); ok { // 递归解析err是否实现Unwrap() error接口
			sb.WriteString("\n" + strings.Repeat("  ", i))
			err = e.Unwrap()
			continue
		}
		break
	}

	sb.WriteString("\n")
	sb.Write(e.Stack())

	// write stack trace to writer
	_, _ = io.Copy(writer, strings.NewReader(sb.String()))
}
