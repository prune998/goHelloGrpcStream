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
	"encoding/json"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	kitlog "github.com/go-kit/kit/log"
	"github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/namsral/flag"
	"github.com/prometheus/client_golang/prometheus"
	pb "github.com/prune998/goHelloGrpcStream/helloworld/helloworld"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var (
	freq    = flag.Duration("freq", 10*time.Second, "frequency for sending a msg")
	debug   = flag.Bool("debug", false, "display debugs")
	port    = flag.String("port", "localhost:7788", "port to bind")
	version = "no version set"
)

// server is used to implement helloworld.GreeterServer.
type server struct {
	kitlog.Logger
}

// SayHello implements helloworld.GreeterServer
func (s *server) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
	PromSayHelloReceivedCounter.Inc()
	s.Log("msg", "got request", "client", in.Name, "port", *port)
	return &pb.HelloReply{Message: "Hello " + in.Name + " " + *port}, nil
}

// SayHelloStream implements helloworld.GreeterServer
func (s *server) SayHelloStream(stream pb.Greeter_SayHelloStreamServer) error {
	PromSayHelloStreamReceivedCounter.Inc()
	for {
		select {
		case <-time.After(5 * time.Second):
			err := stream.Send(&pb.HelloReply{Message: "Hello Stream " + *port})
			if err == io.EOF {
				s.Log("msg", "EOF while sending alerts to user", "err", err)
				break
			}
			if err != nil {
				s.Log("msg", "Error while sending alerts to user", "err", err)
				break
			}
		case <-stream.Context().Done():
			return nil
		}

	}
}

func main() {
	flag.Parse()

	// setup logger with Json output
	logger := kitlog.NewJSONLogger(kitlog.NewSyncWriter(os.Stdout))
	logger = kitlog.With(logger, "application", "greeter_server", "ts", kitlog.DefaultTimestampUTC, "caller", kitlog.DefaultCaller)

	lis, err := net.Listen("tcp", *port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpc_prometheus.EnableHandlingTimeHistogram()
	serverOpts := []grpc.ServerOption{
		grpc.StreamInterceptor(grpc_prometheus.StreamServerInterceptor),
		grpc.UnaryInterceptor(grpc_prometheus.UnaryServerInterceptor),
	}
	s := grpc.NewServer(serverOpts...)
	pb.RegisterGreeterServer(s, &server{logger})
	// Register reflection service on gRPC server.
	reflection.Register(s)

	// healthz basic
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		m := map[string]interface{}{"version": version, "status": "OK"}

		b, err := json.Marshal(m)
		if err != nil {
			http.Error(w, "no valid point for this device_id", 500)
			return
		}

		w.Write(b)
	})

	// prometheus metrics
	http.Handle("/metrics", prometheus.Handler())

	logger.Log("msg", "Listening on tcp://localhost:"+*port)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
