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
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus"
	"github.com/grpc-ecosystem/go-grpc-middleware/tags"
	"github.com/grpc-ecosystem/go-grpc-prometheus"

	"github.com/namsral/flag"
	"github.com/prometheus/client_golang/prometheus"
	pb "github.com/prune998/goHelloGrpcStream/helloworld/helloworld"
	"github.com/sirupsen/logrus"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

var (
	freq     = flag.Duration("freq", 10*time.Second, "frequency for sending a msg")
	debug    = flag.Bool("debug", false, "display debugs")
	grpcPort = flag.String("grpcport", "7788", "port to bind for GRPC")
	httpPort = flag.String("httpport", "7789", "port to bind for HTTP")
	version  = "no version set"
)

// server is used to implement helloworld.GreeterServer.
type server struct {
	*logrus.Logger
}

// SayHello implements helloworld.GreeterServer
func (s *server) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
	log := s.WithFields(logrus.Fields{
		"client":   in.Name,
		"port":     *grpcPort,
		"endpoint": "SayHello",
	})
	PromSayHelloReceivedCounter.Inc()
	log.Infof("got request from client %v:%v", in.Name, *grpcPort)
	return &pb.HelloReply{Message: "Hello " + in.Name + " " + *grpcPort}, nil
}

// SayHelloStream implements helloworld.GreeterServer
func (s *server) SayHelloStream(stream pb.Greeter_SayHelloStreamServer) error {
	log := s.WithFields(logrus.Fields{
		"port":     *grpcPort,
		"endpoint": "SayHelloStream",
	})
	PromSayHelloStreamReceivedCounter.Inc()
	PromSayHelloStreamReceivedGauge.Inc()
	log.Info("SayHelloStream called")
	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			log.Errorf("EOF while sending alerts to user: %v", err)
			PromSayHelloStreamReceivedGauge.Dec()
			break
		}
		if err != nil {
			log.Errorf("Error while sending alerts to user: %v", err)
			PromSayHelloStreamReceivedGauge.Dec()
			break
		}

		log.Infof("Reveived Stream message %v", msg.Name)

		// we reply to the message
		err = stream.Send(&pb.HelloReply{Message: "Pong " + msg.Name})
		if err == io.EOF {
			log.Errorf("EOF while sending alerts to user: %v", err)
			PromSayHelloStreamReceivedGauge.Dec()
			break
		}
		if err != nil {
			log.Errorf("Error while sending alerts to user: %v", err)
			PromSayHelloStreamReceivedGauge.Dec()
			break
		}

	}
	return nil
}

func main() {
	flag.Parse()

	// Logrus
	// Log as JSON instead of the default ASCII formatter.
	logrus.SetFormatter(&logrus.JSONFormatter{})

	// Output to stdout instead of the default stderr
	// Can be any io.Writer, see below for File example
	logrus.SetOutput(os.Stdout)

	// Only log the warning severity or above.
	if *debug {
		logrus.SetLevel(logrus.DebugLevel)
	} else {
		logrus.SetLevel(logrus.WarnLevel)
	}

	// all the above logrus is not working...
	// now we create the logrus logger
	logger := logrus.New()
	log := logger.WithFields(logrus.Fields{
		"application": "greeter_server",
	})
	grpc_logrus.ReplaceGrpcLogger(log)
	lis, err := net.Listen("tcp", fmt.Sprintf(":%v", *grpcPort))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	// add logrus accesslogs
	opts := []grpc_logrus.Option{
		grpc_logrus.WithDurationField(func(duration time.Duration) (key string, value interface{}) {
			return "grpc.time_ns", duration.Nanoseconds()
		}),
	}

	// configure the gRPC endpoint to report metrics, logs and increase HTTP2 streams
	grpc_prometheus.EnableHandlingTimeHistogram()
	serverOpts := []grpc.ServerOption{
		grpc.MaxConcurrentStreams(50000),
		grpc_middleware.WithUnaryServerChain(
			grpc_ctxtags.UnaryServerInterceptor(),
			grpc_logrus.UnaryServerInterceptor(log, opts...),
		),
		grpc_middleware.WithStreamServerChain(
			grpc_ctxtags.StreamServerInterceptor(),
			grpc_logrus.StreamServerInterceptor(log, opts...),
		),
	}
	s := grpc.NewServer(serverOpts...)
	pb.RegisterGreeterServer(s, &server{logger})

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

	// listen on the HTTP port for metrics
	go func() {
		log.Warn(fmt.Sprintf("listening HTTP (metrics & map) on %v", *httpPort))
		log.Warn(http.ListenAndServe(fmt.Sprintf(":%s", *httpPort), nil))
	}()

	// start the gRPC port
	log.Warnf("Listening on tcp://localhost:%v", *grpcPort)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
