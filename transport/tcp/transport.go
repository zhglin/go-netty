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

package tcp

import (
	"net"

	"github.com/go-netty/go-netty/transport"
)

// tcp客户端链接
type tcpTransport struct {
	*net.TCPConn
}

func (t *tcpTransport) Writev(buffs transport.Buffers) (int64, error) {
	return buffs.Buffers.WriteTo(t.TCPConn)
}

func (t *tcpTransport) Flush() error {
	return nil
}

func (t *tcpTransport) RawTransport() interface{} {
	return t.TCPConn
}

// 设置tcp客户端链接配置选项
func (t *tcpTransport) applyOptions(tcpOptions *Options, client bool) (*tcpTransport, error) {

	// SetKeepAlive设置操作系统是否应该在连接上发送keepalive消息。
	if err := t.SetKeepAlive(tcpOptions.KeepAlive); nil != err {
		return t, err
	}

	// SetKeepAlivePeriod设置keepalive的周期。
	if err := t.SetKeepAlivePeriod(tcpOptions.KeepAlivePeriod); nil != err {
		return t, err
	}

	// SetLinger设置一个连接在有数据等待发送或确认时的关闭行为。
	//如果sec < 0(默认值)，则操作系统在后台完成数据发送。
	//如果sec == 0，则操作系统丢弃所有未发送或未确认的数据。
	//如果sec > 0，则数据在后台发送，因为sec < 0。在某些操作系统上，经过几秒之后，任何剩余未发送的数据都可能被丢弃。
	if err := t.SetLinger(tcpOptions.Linger); nil != err {
		return t, err
	}

	// SetNoDelay控制操作系统是否应该延迟数据包传输以希望发送更少的数据包(Nagle算法)。默认值为true(无延迟)，这意味着数据在写入后会尽快发送。
	if err := t.SetNoDelay(tcpOptions.NoDelay); nil != err {
		return t, err
	}

	if tcpOptions.SockBuf > 0 {
		// SetReadBuffer设置与连接相关的操作系统接收缓冲区的大小。
		if err := t.SetReadBuffer(tcpOptions.SockBuf); nil != err {
			return t, err
		}

		// SetWriteBuffer设置与连接相关的操作系统传输缓冲区的大小。
		if err := t.SetWriteBuffer(tcpOptions.SockBuf); nil != err {
			return t, err
		}
	}

	return t, nil
}
