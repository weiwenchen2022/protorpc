# protorpc

Package protorpc implements a Protobuf-RPC ClientCodec and ServerCodec for the rpc package.

# Install

Install Go plugins for the protocol compiler:

`go install google.golang.org/protobuf/cmd/protoc-gen-go@latest`

`go install github.com/weiwenchen2022/protorpc/cmd/protoc-gen-go-netrpc@latest`

Install `protorpc` package:

`go get github.com/weiwenchen2022/protorpc`

# Examples

First, Open [helloworld/helloworld.proto](examples/helloworld/helloworld.proto):

```Proto
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
```

Second, compile the helloworld.proto file, run the following command (we can use `go generate` to invoke this command, see [helloworld.go](examples/helloworld/helloworld/helloworld.go)): 

	protoc --go_out=. --go_opt=paths=source_relative \
    	--go-netrpc_out=. --go-netrpc_opt=paths=source_relative \
    	helloworld/helloworld.proto

This will regenerate the helloworld.pb.go and helloworld_netrpc.pb.go files.

Now, we can use the generated code like this:

```Go
package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/rpc"

	"github.com/weiwenchen2022/protorpc"
	pb "github.com/weiwenchen2022/protorpc/examples/helloworld/helloworld"
)

// server is used to implement helloworld.GreeterServer.
type server struct {
	pb.UnimplementedGreeterServer
}

// SayHello implements helloworld.GreeterServer
func (s *server) SayHello(args *pb.HelloRequest, reply *pb.HelloReply) error {
	log.Printf("Received: %v", args.GetName())
	*reply = pb.HelloReply{Message: "Hello " + args.GetName()}
	return nil
}

func main() {
	l, e := net.Listen("tcp", ":1234")
	if e != nil {
		log.Fatal("listen error:", e)
	}

	s := rpc.NewServer()
	pb.RegisterGreeterServer(s, new(server))

	log.Printf("server listening at %v", l.Addr())
	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				log.Fatalf("failed to accept: %v", err)
			}
			go s.ServeCodec(protorpc.NewServerCodec(conn))
		}
	}()

	conn, err := protorpc.Dial("tcp", "localhost:1234")
	if err != nil {
		log.Fatal("dialing:", err)
	}
	defer conn.Close()

	c := pb.NewGreeterClient(conn)

	// Synchronous call
	args := &pb.HelloRequest{Name: "world"}
	var reply pb.HelloReply
	err = c.SayHello(args, &reply)
	if err != nil {
		log.Fatal("greet error:", err)
	}
	fmt.Printf("Greeting: %s", reply.GetMessage())

	// Asynchronous call
	helloReply := new(pb.HelloReply)
	helloCall := c.SayHelloAsync(args, helloReply, nil)
	helloCall = <-helloCall.Done // will be equal to helloCall
	// check errors, print, etc.
}
```

# Reference

`godoc` or [http://godoc.org/github.com/weiwenchen2022/protorpc](http://godoc.org/github.com/weiwenchen2022/protorpc)
