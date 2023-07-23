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

var (
	port = flag.Int("port", 50051, "The server port")
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
	flag.Parse()

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := rpc.NewServer()
	pb.RegisterGreeterServer(s, new(server))

	log.Printf("server listening at %v", lis.Addr())
	for {
		conn, err := lis.Accept()
		if err != nil {
			log.Fatalf("failed to accept: %v", err)
		}
		go s.ServeCodec(protorpc.NewServerCodec(conn))
	}
}
