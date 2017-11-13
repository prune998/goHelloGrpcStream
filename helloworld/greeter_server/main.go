/*
 *
 * Copyright 2015 gRPC authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

//go:generate protoc -I ../helloworld --go_out=plugins=grpc:../helloworld ../helloworld/helloworld.proto

package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"time"

	pb "github.com/prune998/goHelloGrpcStream/helloworld/helloworld"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

const (
	port = ":7788"
)

// server is used to implement helloworld.GreeterServer.
type server struct{}

// SayHello implements helloworld.GreeterServer
func (s *server) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
	fmt.Printf("got request for %s on port %v\n", in.Name, port)
	return &pb.HelloReply{Message: "Hello " + in.Name + port}, nil
}

// SayHelloStream implements helloworld.GreeterServer
func (s *server) SayHelloStream(hello *pb.HelloRequest, stream pb.Greeter_SayHelloStreamServer) error {
	for {
		select {
		case <-time.After(5 * time.Second):
			err := stream.Send(&pb.HelloReply{Message: "Hello Stream " + hello.Name + port})
			if err == io.EOF {
				fmt.Printf("err", "EOF while sending alerts to user")
				break
			}
			if err != nil {
				fmt.Printf("err", "EOF while sending alerts to user")
				break
			}
		case <-stream.Context().Done():
			return nil
		}

	}
}

func main() {
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterGreeterServer(s, &server{})
	// Register reflection service on gRPC server.
	reflection.Register(s)
	log.Println("Listening on tcp://localhost:", port)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
