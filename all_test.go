// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package protorpc

import (
	"errors"
	"io"
	"net"
	"net/rpc"
	"reflect"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/weiwenchen2022/protorpc/internal/stream"
	pb "github.com/weiwenchen2022/protorpc/protorpcpb"
	testpb "github.com/weiwenchen2022/protorpc/testpb"

	"google.golang.org/protobuf/types/known/anypb"
)

type Arith int

func (*Arith) Add(args *testpb.Args, reply *testpb.Reply) error {
	reply.C = args.A + args.B
	return nil
}

func (*Arith) Mul(args *testpb.Args, reply *testpb.Reply) error {
	reply.C = args.A * args.B
	return nil
}

func (*Arith) Div(args *testpb.Args, reply *testpb.Reply) error {
	if args.B == 0 {
		return errors.New("divide by zero")
	}
	reply.C = args.A / args.B
	return nil
}

type BuiltinTypes struct{}

func (BuiltinTypes) Map(args *testpb.Args, reply *testpb.Map) error {
	if reply.M == nil {
		reply.M = make(map[int64]int64)
	}
	reply.M[args.A] = args.B
	return nil
}

func (BuiltinTypes) Slice(args *testpb.Args, reply *testpb.Slice) error {
	reply.S = append(reply.S, args.A)
	return nil
}

func init() {
	rpc.Register(new(Arith))
	rpc.Register(BuiltinTypes{})
}

func TestServerNoParam(t *testing.T) {
	cli, srv := net.Pipe()
	defer cli.Close()
	go ServeConn(srv)
	dec := stream.NewDecoder(cli)
	enc := stream.NewEncoder(cli)

	enc.Encode(&pb.Request{Method: "Arith.Add", ID: 123})
	var resp pb.Response
	if err := dec.Decode(&resp); err != nil {
		t.Fatalf("Decode after no param: %v", err)
	}
	if resp.Error == "" {
		t.Fatalf("Expected error, got empty")
	}
}

func TestServerEmptyMessage(t *testing.T) {
	cli, srv := net.Pipe()
	defer cli.Close()
	go ServeConn(srv)
	dec := stream.NewDecoder(cli)
	enc := stream.NewEncoder(cli)

	enc.Encode(&pb.Request{})
	var resp pb.Response
	if err := dec.Decode(&resp); err != nil {
		t.Fatalf("Decode after empty: %v", err)
	}
	if resp.Error == "" {
		t.Fatalf("Expected error, got empty")
	}
}

func TestServer(t *testing.T) {
	cli, srv := net.Pipe()
	defer cli.Close()
	go ServeConn(srv)
	dec := stream.NewDecoder(cli)
	enc := stream.NewEncoder(cli)

	// Send hand-coded requests to server, parse responses.
	for i := 0; i < 10; i++ {
		var req = pb.Request{Method: "Arith.Add", ID: uint64(i)}
		req.Param, _ = anypb.New(&testpb.Args{A: int64(i), B: int64(i + 1)})
		enc.Encode(&req)
		var resp pb.Response
		err := dec.Decode(&resp)
		if err != nil {
			t.Fatalf("Decode: %v", err)
		}
		if resp.Error != "" {
			t.Fatalf("resp.Error: %s", resp.Error)
		}
		if uint64(i) != resp.ID {
			t.Errorf("resp: bad id %v want %d", resp.ID, uint64(i))
		}
		var reply testpb.Reply
		if err := resp.Result.UnmarshalTo(&reply); err != nil {
			t.Fatal("UnmarshalTo error:", err)
		}
		if 2*i+1 != int(reply.C) {
			t.Errorf("resp: bad result: %d + %d = %d", i, i+1, reply.C)
		}
	}
}

func TestClient(t *testing.T) {
	// Assume server is okay (TestServer is above).
	// Test client against server.
	cli, srv := net.Pipe()
	go ServeConn(srv)

	client := NewClient(cli)
	defer client.Close()

	// Synchronous calls
	args := &testpb.Args{A: 7, B: 8}
	reply := new(testpb.Reply)
	err := client.Call("Arith.Add", args, reply)
	if err != nil {
		t.Errorf("Add: expected no error but got string %q", err)
	}
	if args.A+args.B != reply.C {
		t.Errorf("Add: got %d expected %d", reply.C, args.A+args.B)
	}

	args = &testpb.Args{A: 7, B: 8}
	reply = new(testpb.Reply)
	err = client.Call("Arith.Mul", args, reply)
	if err != nil {
		t.Errorf("Mul: expected no error but got string %q", err)
	}
	if args.A*args.B != reply.C {
		t.Errorf("Mul: got %d expected %d", reply.C, args.A*args.B)
	}

	// Out of order.
	args = &testpb.Args{A: 7, B: 8}
	mulReply := new(testpb.Reply)
	mulCall := client.Go("Arith.Mul", args, mulReply, nil)
	addReply := new(testpb.Reply)
	addCall := client.Go("Arith.Add", args, addReply, nil)

	addCall = <-addCall.Done
	if addCall.Error != nil {
		t.Errorf("Add: expected no error but got string %q", addCall.Error)
	}
	if args.A+args.B != addReply.C {
		t.Errorf("Add: got %d expected %d", addReply.C, args.A+args.B)
	}

	mulCall = <-mulCall.Done
	if mulCall.Error != nil {
		t.Errorf("Mul: expected no error but got string %q", mulCall.Error)
	}
	if args.A*args.B != mulReply.C {
		t.Errorf("Mul: got %d expected %d", mulReply.C, args.A*args.B)
	}

	// Error test
	args = &testpb.Args{A: 7, B: 0}
	reply = new(testpb.Reply)
	err = client.Call("Arith.Div", args, reply)
	// expect an error: zero divide
	if err == nil {
		t.Error("Div: expected error")
	} else if err.Error() != "divide by zero" {
		t.Error("Div: expected divide by zero error; got", err)
	}
}

func TestBuiltinTypes(t *testing.T) {
	cli, srv := net.Pipe()
	go ServeConn(srv)

	client := NewClient(cli)
	defer client.Close()

	// Map
	args := &testpb.Args{A: 7, B: 8}
	replyMap := testpb.Map{}
	err := client.Call("BuiltinTypes.Map", args, &replyMap)
	if err != nil {
		t.Errorf("Map: expected no error but got string %q", err)
	}
	if args.B != replyMap.M[args.A] {
		t.Errorf("Map: expected %d got %d", args.B, replyMap.M[args.A])
	}

	// Slice
	var replySlice = testpb.Slice{}
	err = client.Call("BuiltinTypes.Slice", args, &replySlice)
	if err != nil {
		t.Errorf("Slice: expected no error but got string %q", err)
	}
	if e := []int64{args.A}; !reflect.DeepEqual(e, replySlice.S) {
		t.Errorf("Slice: expected %v got %v", e, replySlice.S)
	}
}

func TestMalformedOutput(t *testing.T) {
	cli, srv := net.Pipe()
	go func() { stream.NewEncoder(srv).Encode(&pb.Response{ID: 0}) }()
	go io.ReadAll(srv)

	client := NewClient(cli)
	defer client.Close()

	args := &testpb.Args{A: 7, B: 8}
	reply := new(testpb.Reply)
	err := client.Call("Arith.Add", args, reply)
	if err == nil {
		t.Error("expected error")
	}
}

func TestUnexpectedError(t *testing.T) {
	cli, srv := myPipe()
	go cli.PipeWriter.CloseWithError(errors.New("unexpected error!")) // reader will get this error
	ServeConn(srv)                                                    // must return, not loop
}

// Copied from package net.
func myPipe() (*pipe, *pipe) {
	r1, w1 := io.Pipe()
	r2, w2 := io.Pipe()
	return &pipe{r1, w2}, &pipe{r2, w1}
}

type pipe struct {
	*io.PipeReader
	*io.PipeWriter
}

func (p *pipe) Close() error {
	err := p.PipeReader.Close()
	err1 := p.PipeWriter.Close()
	if err == nil {
		err = err1
	}
	return err
}

type clientInterface interface {
	Call(serviceMethod string, args, reply any) error

	Close() error
}

type bench struct {
	perG func(b *testing.B, pb *testing.PB, client clientInterface)
}

func benchrpc(b *testing.B, bench bench) {
	b.Run("gobrpc", func(b *testing.B) {
		_ = rpc.Register(new(Arith))

		cli, srv := net.Pipe()
		go rpc.ServeConn(srv)

		client := rpc.NewClient(cli)
		defer client.Close()

		b.ResetTimer()

		b.RunParallel(func(pb *testing.PB) {
			bench.perG(b, pb, client)
		})
	})

	b.Run("protorpc", func(b *testing.B) {
		cli, srv := net.Pipe()
		go ServeConn(srv)

		client := NewClient(cli)
		defer client.Close()

		b.ResetTimer()

		b.RunParallel(func(pb *testing.PB) {
			bench.perG(b, pb, client)
		})
	})
}

func BenchmarkEndToEnd(b *testing.B) {
	args := &testpb.Args{A: 7, B: 8}

	benchrpc(b, bench{
		perG: func(b *testing.B, p *testing.PB, client clientInterface) {
			reply := new(testpb.Reply)
			for p.Next() {
				// Synchronous calls
				err := client.Call("Arith.Add", args, reply)
				if err != nil {
					b.Fatalf("rpc error: Add: expected no error but got string %q", err)
				}
				if args.A+args.B != reply.C {
					b.Fatalf("rpc error: Add: expected %d got %d", args.A+args.B, reply.C)
				}
			}
		},
	})
}

func BenchmarkEndToEndAsync(b *testing.B) {
	if b.N == 0 {
		return
	}

	const MaxConcurrentCalls = 100

	cli, srv := net.Pipe()
	go ServeConn(srv)

	client := NewClient(cli)
	defer client.Close()

	// Asynchronous calls
	args := &testpb.Args{A: 7, B: 8}
	procs := 4 * runtime.GOMAXPROCS(-1)
	send := int32(b.N)
	recv := int32(b.N)
	var wg sync.WaitGroup
	wg.Add(procs)
	gate := make(chan bool, MaxConcurrentCalls)
	res := make(chan *rpc.Call, MaxConcurrentCalls)
	b.ResetTimer()

	for p := 0; p < procs; p++ {
		go func() {
			for atomic.AddInt32(&send, -1) >= 0 {
				gate <- true
				reply := new(testpb.Reply)
				client.Go("Arith.Add", args, reply, res)
			}
		}()
		go func() {
			for call := range res {
				A := call.Args.(*testpb.Args).A
				B := call.Args.(*testpb.Args).B
				C := call.Reply.(*testpb.Reply).C
				if A+B != C {
					b.Errorf("incorrect reply: Add: expected %d got %d", A+B, C)
					return
				}
				<-gate
				if atomic.AddInt32(&recv, -1) == 0 {
					close(res)
				}
			}
			wg.Done()
		}()
	}
	wg.Wait()
}
