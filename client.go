// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package protorpc

import (
	"io"
	"net"
	"net/rpc"

	"github.com/weiwenchen2022/protorpc/internal/maps"
	"github.com/weiwenchen2022/protorpc/internal/stream"
	pb "github.com/weiwenchen2022/protorpc/protorpcpb"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

type clientCodec struct {
	dec *stream.Decoder // for reading proto buffer values
	enc *stream.Encoder // for writing proto buffer values
	c   io.Closer

	// temporary work space
	req  pb.Request
	resp pb.Response

	// Proto-RPC responses include the request id but not the request method.
	// Package rpc expects both.
	// We save the request method in pending when sending a request
	// and then look it up by request ID when filling out the rpc Response.
	pending maps.Map[uint64, string] // map request id to method name
}

// NewClientCodec returns a new rpc.ClientCodec using Proto-RPC on conn.
func NewClientCodec(conn io.ReadWriteCloser) rpc.ClientCodec {
	return &clientCodec{
		dec: stream.NewDecoder(conn),
		enc: stream.NewEncoder(conn),
		c:   conn,
	}
}

func (c *clientCodec) WriteRequest(r *rpc.Request, param any) error {
	c.pending.Store(r.Seq, r.ServiceMethod)
	c.req.Method = r.ServiceMethod

	var err error
	if c.req.Param, err = anypb.New(param.(proto.Message)); err != nil {
		return err
	}
	c.req.ID = r.Seq
	return c.enc.Encode(&c.req)
}

func (c *clientCodec) ReadResponseHeader(r *rpc.Response) error {
	c.resp.Reset()
	if err := c.dec.Decode(&c.resp); err != nil {
		return err
	}

	r.ServiceMethod, _ = c.pending.LoadAndDelete(c.resp.ID)

	r.Error = ""
	r.Seq = c.resp.ID
	if c.resp.Error != "" || c.resp.Result == nil {
		x := c.resp.Error
		if x == "" {
			x = "unspecified error"
		}
		r.Error = x
	}
	return nil
}

func (c *clientCodec) ReadResponseBody(x any) error {
	if x == nil {
		return nil
	}
	return c.resp.Result.UnmarshalTo(x.(proto.Message))
}

func (c *clientCodec) Close() error {
	return c.c.Close()
}

// NewClient returns a new rpc.Client to handle requests to the
// set of services at the other end of the connection.
func NewClient(conn io.ReadWriteCloser) *rpc.Client {
	return rpc.NewClientWithCodec(NewClientCodec(conn))
}

// Dial connects to a Proto-RPC server at the specified network address.
func Dial(network, address string) (*rpc.Client, error) {
	conn, err := net.Dial(network, address)
	if err != nil {
		return nil, err
	}
	return NewClient(conn), nil
}
