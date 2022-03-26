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

	"github.com/go-netty/go-netty/utils"
)

// Pipeline defines a message processing pipeline.
// 定义消息处理流水线。
type Pipeline interface {

	// AddFirst add a handler to the first.
	// 头部添加handler
	AddFirst(handlers ...Handler) Pipeline

	// AddLast add a handler to the last.
	// 尾部添加handler
	AddLast(handlers ...Handler) Pipeline

	// AddHandler add handlers in position.
	// 指定位置添加handler
	AddHandler(position int, handlers ...Handler) Pipeline

	// IndexOf find fist index of handler.
	// 通过比较函数返回指定handler所在下标
	IndexOf(func(Handler) bool) int

	// LastIndexOf find last index of handler.
	LastIndexOf(func(Handler) bool) int

	// ContextAt get context by position.
	ContextAt(position int) HandlerContext

	// Size of handler
	Size() int

	// Channel get channel.
	Channel() Channel

	// ServeChannel serve the channel.
	// 关联channel 启动读写
	ServeChannel(channel Channel)

	FireChannelActive()
	// FireChannelRead 从head开始执行读的handler  message可以是个链接 也可以是从链接中读取到的消息
	FireChannelRead(message Message)
	FireChannelWrite(message Message)
	// FireChannelException 链接异常的处理handler
	FireChannelException(ex Exception)
	FireChannelInactive(ex Exception)
	FireChannelEvent(event Event)
}

// NewPipeline convert to PipelineFactory
// 创建消息处理流水线
func NewPipeline() PipelineFactory {
	return NewPipelineWith
}

// NewPipelineWith create a pipeline.
// 创建流水线
func NewPipelineWith() Pipeline {

	p := &pipeline{}

	// 设置消息头的handler
	p.head = &handlerContext{
		pipeline: p,
		handler:  new(headHandler),
	}

	p.tail = &handlerContext{
		pipeline: p,
		handler:  new(tailHandler),
	}

	p.head.next = p.tail
	p.tail.prev = p.head

	// head + tail
	p.size = 2
	return p
}

// pipeline to implement Pipeline
type pipeline struct {
	head    *handlerContext // 头
	tail    *handlerContext // 尾
	channel Channel         // 关联的channel
	size    int             // 流水线中handler中的数量
}

// AddFirst to add handlers at head 头部添加handler
func (p *pipeline) AddFirst(handlers ...Handler) Pipeline {
	// checking handler.
	checkHandler(handlers...)

	for _, h := range handlers {
		p.addFirst(h)
	}
	return p
}

// AddLast to add handlers at tail 尾部添加
func (p *pipeline) AddLast(handlers ...Handler) Pipeline {
	// checking handler.
	checkHandler(handlers...)

	for _, h := range handlers {
		p.addLast(h)
	}
	return p
}

// AddHandler to insert handlers in position 指定位置添加handler
func (p *pipeline) AddHandler(position int, handlers ...Handler) Pipeline {

	// checking handler.
	checkHandler(handlers...)

	// checking position.
	utils.AssertIf(position >= p.size, "invalid position: %d", position)

	if -1 == position || position == p.size-1 {
		return p.AddLast(handlers...)
	}

	// 找到i位置的node
	curNode := p.head
	for i := 0; i < position; i++ {
		curNode = curNode.next
	}

	for _, h := range handlers {
		oldNext := curNode.next
		curNode.next = &handlerContext{
			pipeline: p,
			handler:  h,
			prev:     curNode,
			next:     oldNext,
		}

		oldNext.prev = curNode.next
		curNode = curNode.next
		p.size++
	}

	return p
}

// IndexOf to find fist index of handler.
func (p *pipeline) IndexOf(comp func(Handler) bool) int {

	head := p.head

	for i := 0; ; i++ {
		if comp(head.handler) {
			return i
		}
		if head = head.next; head == nil {
			break
		}
	}

	return -1
}

// LastIndexOf to find last index of handler.
func (p *pipeline) LastIndexOf(comp func(Handler) bool) int {

	tail := p.tail

	for i := p.size - 1; ; i-- {
		if comp(tail.handler) {
			return i
		}
		if tail = tail.prev; tail == nil {
			break
		}
	}

	return -1
}

// ContextAt to access the context by position
func (p *pipeline) ContextAt(position int) HandlerContext {

	if -1 == position || position >= p.size {
		return nil
	}

	curNode := p.head
	for i := 0; i < position; i++ {
		curNode = curNode.next
	}

	return curNode
}

// Size of handlers
func (p *pipeline) Size() int {
	return p.size
}

// addFirst to add handlers head
func (p *pipeline) addFirst(handler Handler) {

	oldNext := p.head.next
	p.head.next = &handlerContext{
		pipeline: p,
		handler:  handler,
		prev:     p.head,
		next:     oldNext,
	}

	oldNext.prev = p.head.next
	p.size++
}

// addLast to add handlers tail
func (p *pipeline) addLast(handler Handler) {

	oldPrev := p.tail.prev
	p.tail.prev = &handlerContext{
		pipeline: p,
		handler:  handler,
		prev:     oldPrev,
		next:     p.tail,
	}

	oldPrev.next = p.tail.prev
	p.size++
}

// Channel to get channel of Pipeline
func (p *pipeline) Channel() Channel {
	return p.channel
}

// ServeChannel serveChannel to serve the channel
// 关联channel
func (p *pipeline) ServeChannel(channel Channel) {

	utils.AssertIf(nil != p.channel, "already attached channel")
	p.channel = channel
	p.channel.serveChannel()
}

func (p *pipeline) FireChannelActive() {
	p.head.HandleActive()
}

// FireChannelRead 从head开始执行读的handler
func (p *pipeline) FireChannelRead(message Message) {
	p.head.HandleRead(message)
}

func (p *pipeline) FireChannelWrite(message Message) {
	p.tail.HandleWrite(message)
}

// FireChannelException 链接异常的处理handler
func (p *pipeline) FireChannelException(ex Exception) {
	p.head.HandleException(ex)
}

func (p *pipeline) FireChannelInactive(ex Exception) {
	p.head.HandleInactive(ex)
}

func (p *pipeline) FireChannelEvent(event Event) {
	p.head.HandleEvent(event)
}

// checkHandler to checking handlers
// 校验handler是否合法
func checkHandler(handlers ...Handler) {

	for index, h := range handlers {
		switch h.(type) {
		case ActiveHandler:
		case InboundHandler:
		case OutboundHandler:
		case ExceptionHandler:
		case InactiveHandler:
		case EventHandler:
		default:
			utils.Assert(fmt.Errorf("unrecognized Handler: %d:%T", index, h))
		}
	}
}
