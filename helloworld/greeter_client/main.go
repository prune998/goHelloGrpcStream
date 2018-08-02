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

package main

import (
	"crypto/tls"
	"io"
	"os"

	"github.com/grpc-ecosystem/go-grpc-prometheus"

	kitlog "github.com/go-kit/kit/log"
	"github.com/namsral/flag"
	pb "github.com/prune998/goHelloGrpcStream/helloworld/helloworld"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const (
	defaultName = "world"
)

var (
	debug              = flag.Bool("debug", false, "display debugs")
	server             = flag.String("server", "localhost:7788", "Greeter Server URL")
	name               = flag.String("name", "world", "name of the client (will be displayed in the server)")
	unary              = flag.Bool("unary", false, "open unary HTTP/2 connextion")
	stream             = flag.Bool("stream", false, "open stream HTTP/2 connection")
	withTLS            = flag.Bool("tls", false, "whether to use TLS")
	insecureSkipVerify = flag.Bool("insecureSkipVerify", true, "whether to ignore security checks")
)

func main() {
	flag.Parse()

	// setup logger with Json output
	logger := kitlog.NewJSONLogger(kitlog.NewSyncWriter(os.Stdout))
	logger = kitlog.With(logger, "application", "greeter_server", "ts", kitlog.DefaultTimestampUTC, "caller", kitlog.DefaultCaller)

	// Setup gRPC options and TLS
	grpcOpts := []grpc.DialOption{
		grpc.WithUnaryInterceptor(grpc_prometheus.UnaryClientInterceptor),
		grpc.WithStreamInterceptor(grpc_prometheus.StreamClientInterceptor),
	}

	if *withTLS {
		creds := credentials.NewTLS(&tls.Config{InsecureSkipVerify: *insecureSkipVerify})
		grpcOpts = append(grpcOpts, grpc.WithTransportCredentials(creds))
	} else {
		grpcOpts = append(grpcOpts, grpc.WithInsecure())
	}

	// Set up a connection to the server.
	conn, err := grpc.Dial(*server, grpcOpts...)
	if err != nil {
		logger.Log("msg", "cant connect to server", "err", err)
	}
	defer conn.Close()
	c := pb.NewGreeterClient(conn)

	if *unary {
		logger.Log("msg", "opening unary connection")
		// Contact the server and print out its response.
		r, err := c.SayHello(context.Background(), &pb.HelloRequest{Name: *name})
		if err != nil {
			logger.Log("msg", "could not greet server", "err", err)
		}
		logger.Log("msg", "Received Greeting: "+r.Message)
	}
	if *stream {
		// request for the Stream
		logger.Log("msg", "opening Stream connection")
		stream, err := c.SayHelloStream(context.Background())
		if err != nil {
			logger.Log("msg", "could not greet server using Streams", "err", err)
		}
		// send a message in the stream
		err = stream.SendMsg(&pb.HelloReply{Message: "Ping Client"})
		if err != nil {
			logger.Log("msg", "error while sending ping to server", "err", err)
			os.Exit(1)
		}
		for {
			msg, err := stream.Recv()
			if err == io.EOF {
				logger.Log("msg", "got EOF from server", "err", err)
				break
			}
			if err != nil {
				//log.Fatalf("%v.GetCustomers(_) = _, %v", c, err)
				logger.Log("msg", "got error from server", "err", err)
				break
			}
			logger.Log("msg", msg.Message)
		}
	}
	logger.Log("msg", "done testing gRPC connections")
}
