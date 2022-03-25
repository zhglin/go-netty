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
	"context"
	"fmt"
	"sync"

	"github.com/go-netty/go-netty/transport"
	"github.com/go-netty/go-netty/transport/tcp"
)

// Bootstrap makes it easy to bootstrap a channel
// 外面包装的一层 方便使用
type Bootstrap interface {
	// Context return context
	Context() context.Context
	// Listen create a listener
	Listen(url string, option ...transport.Option) Listener
	// Connect to remote endpoint
	Connect(url string, attachment Attachment, option ...transport.Option) (Channel, error)
	// Shutdown boostrap
	Shutdown()
}

// NewBootstrap create a new Bootstrap with default config.
// 使用默认配置创建一个新的Bootstrap。
func NewBootstrap(option ...Option) Bootstrap {

	opts := &bootstrapOptions{
		channelIDFactory: SequenceID(),
		pipelineFactory:  NewPipeline(),
		channelFactory:   NewChannel(128),
		transportFactory: tcp.New(),
	}
	opts.bootstrapCtx, opts.bootstrapCancel = context.WithCancel(context.Background())

	for i := range option {
		option[i](opts)
	}

	return &bootstrap{bootstrapOptions: opts}
}

// bootstrap implement
type bootstrap struct {
	*bootstrapOptions
	listeners sync.Map // url - Listener 服务器端的listener
}

// Context to get context
func (bs *bootstrap) Context() context.Context {
	return bs.bootstrapCtx
}

// serveTransport to serve channel
// 服务器端处理已建立的客户端链接
func (bs *bootstrap) serveTransport(transport transport.Transport, attachment Attachment, childChannel bool) Channel {

	// create a new pipeline 创建消息处理流水线
	pipeline := bs.pipelineFactory()

	// generate a channel id  id生成器
	cid := bs.channelIDFactory()

	// create a channel 创建channel
	channel := bs.channelFactory(cid, bs.bootstrapCtx, pipeline, transport)

	// set the attachment if necessary 如有必要，设置附带信息
	if nil != attachment {
		channel.SetAttachment(attachment)
	}

	// initialization pipeline 初始化流水线
	if childChannel {
		bs.childInitializer(channel)
	} else {
		bs.clientInitializer(channel)
	}

	// serve channel.
	channel.Pipeline().ServeChannel(channel)
	return channel
}

// Connect to the remote server with options
func (bs *bootstrap) Connect(url string, attachment Attachment, option ...transport.Option) (Channel, error) {

	options, err := transport.ParseOptions(bs.Context(), url, option...)
	if nil != err {
		return nil, err
	}

	// connect to remote endpoint
	t, err := bs.transportFactory.Connect(options)
	if nil != err {
		return nil, err
	}

	// serve client transport
	return bs.serveTransport(t, attachment, false), nil
}

// Listen to the address with options
// 服务器端进行lister
func (bs *bootstrap) Listen(url string, option ...transport.Option) Listener {
	l := &listener{bs: bs, url: url, option: option}
	bs.listeners.Store(url, l)
	return l
}

// Shutdown the bootstrap
func (bs *bootstrap) Shutdown() {
	bs.bootstrapCancel()

	bs.listeners.Range(func(key, value interface{}) bool {
		value.(Listener).Close()
		return true
	})
}

// removeListener close the listener with url
func (bs *bootstrap) removeListener(url string) {
	bs.listeners.Delete(url)
}

// Listener 服务器端接口
type Listener interface {
	// Close the listener
	// 关闭链接
	Close() error
	// Sync waits for this listener until it is done
	Sync() error
	// Async nonblock waits for this listener
	Async(func(error))
}

// impl Listener
type listener struct {
	bs       *bootstrap
	url      string
	option   []transport.Option
	options  *transport.Options
	acceptor transport.Acceptor // 服务器端监听链接
}

// Close listener
// 服务器端关闭
func (l *listener) Close() error {
	if l.acceptor != nil {
		l.bs.removeListener(l.url)
		return l.acceptor.Close()
	}
	return nil
}

// Sync 服务器端监听并获取客户端链接
func (l *listener) Sync() error {

	// 链接已经建立过了
	if nil != l.acceptor {
		return fmt.Errorf("duplicate call Listener:Sync")
	}

	var err error
	if l.options, err = transport.ParseOptions(l.bs.Context(), l.url, l.option...); nil != err {
		return err
	}

	// 建立服务器端传输层链接
	if l.acceptor, err = l.bs.transportFactory.Listen(l.options); nil != err {
		return err
	}

	for {
		// accept the transport 接收链接
		t, err := l.acceptor.Accept()
		if nil != err {
			return err
		}

		select {
		case <-l.bs.Context().Done():
			// bootstrap has been closed
			return t.Close()
		default:
			// serve child transport
			l.bs.serveTransport(t, nil, true)
		}
	}
}

// Async 异步处理
func (l *listener) Async(fn func(err error)) {
	go func() {
		fn(l.Sync())
	}()
}
