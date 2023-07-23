// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
Package protorpc implements a Protobuf-RPC ClientCodec and ServerCodec
for the rpc package.

Prerequisites:

  - Go, any one of the three latest major releases of Go[https://go.dev/doc/devel/release].

    For installation instructions, see Go’s Getting Started[https://go.dev/doc/install] guide.

  - Protocol buffer[https://protobuf.dev/] compiler, protoc, version 3[https://protobuf.dev/programming-guides/proto3/].

  - Go plugins for the protocol compiler:

    1. Install the protocol compiler plugins for Go using the following commands:

    $ go install google.golang.org/protobuf/cmd/protoc-gen-go@latest

    $ go install github.com/weiwenchen2022/protorpc/cmd/protoc-gen-go-grpc@latest

    2. Update your PATH so that the protoc compiler can find the plugins:

    $ export PATH="$PATH:$(go env GOPATH)/bin"

Here is a simple example. Define a service in a .proto file (helloworld/helloworld.proto):

	syntax = "proto3";

	option go_package = "github.com/weiwenchen2022/protorpc/examples/helloworld/helloworld";

	package helloworld;

	// The greeting service definition.
	service Greeter {
		// Sends a greeting
		rpc SayHello (HelloRequest) returns (HelloReply) {}
	}

	// The request message containing the user's name.
	message HelloRequest {
		string name = 1;
	}

	// The response message containing the greetings
	message HelloReply {
	  string message = 1;
	}

Change to the example directory:

	cd examples/helloworld

Run the following command:

	protoc --go_out=. --go_opt=paths=source_relative \
		--go-netrpc_out=. --go-netrpc_opt=paths=source_relative \
		helloworld/helloworld.proto

This will regenerate the helloworld/helloworld.pb.go and helloworld/helloworld_netrpc.pb.go files,
which contain:

  - Code for populating, serializing, and retrieving HelloRequest and HelloReply message types.
  - Generated client and server code.

The server calls (for TCP service):

	package main

	import (
		"fmt"
		"log"
		"net"
		"net/rpc"

		"github.com/weiwenchen2022/protorpc"
		pb "github.com/weiwenchen2022/protorpc/examples/helloworld/helloworld"
	)

	type server struct {
		pb.UnimplementedGreeterServer
	}

	func (s *server) SayHello(args *pb.HelloRequest, reply *pb.HelloReply) error {
		log.Printf("Received: %v", args.GetName())
		*reply = pb.HelloReply{Message: "Hello " + args.GetName()}
		return nil
	}

	func main() {
		l, e := net.Listen("tcp", ":123")
		if e != nil {
			log.Fatal("listen error:", e)
		}

		s := rpc.NewServer()
		pb.RegisterGreeterServer(s, new(server))

		log.Printf("server listening at %v", l.Addr())
		for {
			conn, err := l.Accept()
			if err != nil {
				log.Fatalf("failed to accept: %v", err)
			}
			go s.ServeCodec(protorpc.NewServerCodec(conn))
		}
	}

At this point, clients can see a service "Greeter" with methods "Greeter.SayHello".
To invoke one, a client first dials the server:

	conn, err := protorpc.Dial("tcp", serverAddress + ":1234")
	if err != nil {
		log.Fatal("dialing:", err)
	}
	defer conn.Close()

	client := pb.NewGreeterClient(conn)

Then it can make a remote call:

	// Synchronous call
	args := &pb.HelloRequest{Name: "world"}
	var reply pb.HelloReply
	err = c.SayHello(args, &reply)
	if err != nil {
		log.Fatal("greet error:", err)
	}
	log.Printf("Greeting: %s", reply.GetMessage())

or

	// Asynchronous call
	helloReply := new(pb.HelloReply)
	helloCall := c.SayHelloAsync(args, helloReply, nil)
	helloCall = <-helloCall.Done // will be equal to helloCall
	// check errors, print, etc.
*/
package protorpc
