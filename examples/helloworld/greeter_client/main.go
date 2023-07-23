package main

import (
	"flag"
	"log"

	"github.com/weiwenchen2022/protorpc"
	pb "github.com/weiwenchen2022/protorpc/examples/helloworld/helloworld"
)

const (
	defaultName = "world"
)

var (
	addr = flag.String("addr", "localhost:50051", "the address to connect to")
	name = flag.String("name", defaultName, "Name to greet")
)

func main() {
	flag.Parse()

	// Set up a connection to the server.
	conn, err := protorpc.Dial("tcp", *addr)
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	c := pb.NewGreeterClient(conn)

	// Contact the server and print out its response.
	// Synchronous call
	var reply pb.HelloReply
	err = c.SayHello(&pb.HelloRequest{Name: *name}, &reply)
	if err != nil {
		log.Fatalf("could not greet: %v", err)
	}
	log.Printf("Greeting: %s", reply.GetMessage())

	// Asynchronous call
	helloReply := new(pb.HelloReply)
	helloCall := c.SayHelloAsync(&pb.HelloRequest{Name: *name}, helloReply, nil)
	helloCall = <-helloCall.Done
	if err = helloCall.Error; err != nil {
		log.Fatalf("could not greet: %v", err)
	}
	log.Printf("Greeting: %s", helloReply.GetMessage())
}
