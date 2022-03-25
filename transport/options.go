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

package transport

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strings"

	"github.com/go-netty/go-netty/utils"
)

// Option defines option function
type Option func(options *Options) error

// Options for transport
// 传输层的配置属性
type Options struct {
	// In server side: listen address.  服务器端:监听地址。
	// In client side: connect address. 客户端:连接地址。
	Address *url.URL

	// other configure pass by context.WithValue
	// 其他配置通过上下文传递
	Context context.Context
}

// AddressWithoutHost convert host:port to :port
// 将"host:port"转换为":port"
func (lo *Options) AddressWithoutHost() string {
	_, port, err := net.SplitHostPort(lo.Address.Host)
	utils.Assert(err)
	return net.JoinHostPort("", port)
}

// Apply options
func (lo *Options) Apply(options ...Option) error {
	for _, option := range options {
		if err := option(lo); nil != err {
			return err
		}
	}
	return nil
}

// ParseOptions parse options
func ParseOptions(ctx context.Context, url string, options ...Option) (*Options, error) {
	option := &Options{Context: ctx}
	return option, option.Apply(append([]Option{withAddress(url)}, options...)...)
}

// withAddress for server listener or client dialer
// withAddress服务器监听器或客户端拨号器的地址
func withAddress(address string) Option {
	return func(options *Options) (err error) {
		if options.Address, err = url.Parse(address); nil != err {
			// compatible host:port
			switch {
			case strings.Contains(err.Error(), "cannot contain colon"):
				options.Address, err = url.Parse(fmt.Sprintf("//%s", address))
			case strings.Contains(err.Error(), "missing protocol scheme"):
				options.Address, err = url.Parse(fmt.Sprintf("//%s", address))
			}
		}
		// default path: /
		if options.Address != nil && "" == options.Address.Path {
			options.Address.Path = "/"
		}
		return err
	}
}

// WithContext to hold other configure pass by context.WithValue
func WithContext(ctx context.Context) Option {
	return func(options *Options) error {
		options.Context = ctx
		return nil
	}
}
