// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package protorpc

import (
	"errors"
	"io"
	"net/rpc"
	"sync/atomic"

	"github.com/weiwenchen2022/protorpc/internal/maps"
	"github.com/weiwenchen2022/protorpc/internal/stream"

	pb "github.com/weiwenchen2022/protorpc/protorpcpb"
	"google.golang.org/protobuf/proto"

	anypb "google.golang.org/protobuf/types/known/anypb"
)

var errMissingParams = missingParamsError{}

type missingParamsError struct{}

func (missingParamsError) Error() string { return "protorpc: request body missing params" }

type serverCodec struct {
	dec *stream.Decoder // for reading proto buffer values
	enc *stream.Encoder // for writing proto buffer values
	c   io.Closer

	// temporary work space
	req pb.Request

	// JSON-RPC clients can use arbitrary json values as request IDs.
	// Package rpc expects uint64 request IDs.
	// We assign uint64 sequence numbers to incoming requests
	// but save the original request ID in the pending map.
	// When rpc responds, we use the sequence number in
	// the response to find the original request ID.
	seq     atomic.Uint64
	pending maps.Map[uint64, uint64]
	closed  atomic.Bool
}

// NewServerCodec returns a new rpc.ServerCodec using Proto-RPC on conn.
func NewServerCodec(conn io.ReadWriteCloser) rpc.ServerCodec {
	return &serverCodec{
		dec: stream.NewDecoder(conn),
		enc: stream.NewEncoder(conn),
		c:   conn,
	}
}

func (c *serverCodec) ReadRequestHeader(r *rpc.Request) error {
	c.req.Reset()
	if err := c.dec.Decode(&c.req); err != nil {
		return err
	}
	r.ServiceMethod = c.req.Method

	r.Seq = c.seq.Add(1)
	c.pending.Store(r.Seq, c.req.ID)
	c.req.ID = 0

	return nil
}

func (c *serverCodec) ReadRequestBody(x any) error {
	if x == nil {
		return nil
	}

	if c.req.Param == nil {
		return errMissingParams
	}

	return c.req.Param.UnmarshalTo(x.(proto.Message))
}

func (c *serverCodec) WriteResponse(r *rpc.Response, x any) error {
	id, ok := c.pending.LoadAndDelete(r.Seq)
	if !ok {
		return errors.New("invalid sequence number in response")
	}

	resp := pb.Response{ID: id}
	if r.Error == "" {
		var err error
		if resp.Result, err = anypb.New(x.(proto.Message)); err != nil {
			return err
		}
	} else {
		resp.Error = r.Error
	}

	return c.enc.Encode(&resp)
}

func (c *serverCodec) Close() error {
	if !c.closed.CompareAndSwap(false, true) {
		return nil
	}
	return c.c.Close()
}

// ServeConn runs the Proto-RPC server on a single connection.
// ServeConn blocks, serving the connection until the client hangs up.
// The caller typically invokes ServeConn in a go statement.
func ServeConn(conn io.ReadWriteCloser) {
	rpc.ServeCodec(NewServerCodec(conn))
}
