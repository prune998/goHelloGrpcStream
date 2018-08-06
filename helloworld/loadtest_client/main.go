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
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/prometheus/client_golang/prometheus"

	kitlog "github.com/go-kit/kit/log"
	"github.com/namsral/flag"
	pb "github.com/prune998/goHelloGrpcStream/helloworld/helloworld"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var (
	debug              = flag.Bool("debug", false, "display debugs")
	server             = flag.String("server", "localhost:7788", "Greeter Server URL")
	name               = flag.String("name", "world", "name of the client (will be displayed in the server)")
	clients            = flag.Int("clients", 1, "number of clients to simulate")
	cnxDelay           = flag.Duration("cnxDelay", 60, "time before closing the HTTP2 Stream in seconds")
	httpPort           = flag.String("httpport", "7789", "port to bind for HTTP")
	version            = "no version set"
	withTLS            = flag.Bool("tls", false, "whether to use TLS")
	insecureSkipVerify = flag.Bool("insecureSkipVerify", true, "whether to ignore security checks")
)

// Client is a worker that will load the server
type Client struct {
	kitlog.Logger
	ID                 string `json:"device_id"`
	debug              bool
	withTLS            bool
	insecureSkipVerify bool
}

// NewClient creates a new client
func NewClient(id string, logger kitlog.Logger, debug, withTLS, insecureSkipVerify bool) *Client {
	if debug {
		logger.Log("msg", "starting client "+id)
	}

	return &Client{
		Logger:             logger,
		ID:                 id,
		debug:              debug,
		withTLS:            withTLS,
		insecureSkipVerify: insecureSkipVerify,
	}
}

// Start a new client
func (c Client) Start(ctx context.Context, jobChan chan<- int, server, name string, id int) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Setup gRPC options and TLS
	grpcOpts := []grpc.DialOption{
		grpc.WithUnaryInterceptor(grpc_prometheus.UnaryClientInterceptor),
		grpc.WithStreamInterceptor(grpc_prometheus.StreamClientInterceptor),
	}

	if c.withTLS {
		creds := credentials.NewTLS(&tls.Config{InsecureSkipVerify: c.insecureSkipVerify})
		grpcOpts = append(grpcOpts, grpc.WithTransportCredentials(creds))
	} else {
		grpcOpts = append(grpcOpts, grpc.WithInsecure())
	}

	conn, err := grpc.Dial(server, grpcOpts...)
	if err != nil {
		c.Logger.Log("msg", "cant connect to server", "err", err, "ID", c.ID)
		jobChan <- id
		return
	}
	defer conn.Close()
	g := pb.NewGreeterClient(conn)

	// This code is removed as we only need to test the Streaming capability
	// Contact the server and print out its response.
	// r, err := g.SayHello(ctx, &pb.HelloRequest{Name: name})
	// if err != nil {
	// 	c.Logger.Log("msg", "could not greet server", "err", err, "ID", c.ID)
	// 	jobChan <- id
	// 	return
	// }
	// PromSayHelloReceivedCounter.Inc()
	// if c.debug {
	// 	c.Logger.Log("msg", "Received Greeting: "+r.Message, "ID", c.ID)
	// }

	// open the stream
	stream, err := g.SayHelloStream(context.Background())
	if err != nil {
		c.Logger.Log("msg", "could not SayHelloStream", "err", err, "ID", c.ID)
		jobChan <- id
		return
	}

	// send a message to the stream
	err = stream.SendMsg(&pb.HelloReply{Message: "Ping " + c.ID})
	if err != nil {
		c.Logger.Log("msg", "error while sending alerts to server", "err", err, "ID", c.ID)
		jobChan <- id
		return
	}
	PromSayHelloStreamGauge.Inc()

	// loop until we are done
	for {
		if c.debug {
			c.Logger.Log("msg", "waiting for server response", "ID", c.ID)
		}

		// blocking call, waiting for the next server response inside the stream
		msg, err := stream.Recv()
		if err == io.EOF {
			c.Logger.Log("msg", "got EOF from server", "err", err, "ID", c.ID)
			PromSayHelloStreamGauge.Dec()
			jobChan <- id
			return
		}
		if err != nil {
			c.Logger.Log("msg", "got error from server", "err", err, "ID", c.ID)
			PromSayHelloStreamGauge.Dec()
			jobChan <- id
			return
		}
		PromSayHelloStreamReceivedCounter.Inc()
		if c.debug {
			c.Logger.Log("msg", msg.Message, "ID", c.ID)
		}

		// we close the stream after a fixed delay specified on the commandline
		time.Sleep(*cnxDelay)

		// closing the stream will send an "EOF from server error"
		err = stream.CloseSend()
		if err != nil {
			c.Logger.Log("msg", "got error from CloseSend", "err", err, "ID", c.ID)
			PromSayHelloStreamGauge.Dec()
			jobChan <- id
			return
		}
	}
}

func main() {
	flag.Parse()

	// setup logger with Json output
	logger := kitlog.NewJSONLogger(kitlog.NewSyncWriter(os.Stdout))
	logger = kitlog.With(logger, "application", "greeter_server", "ts", kitlog.DefaultTimestampUTC, "caller", kitlog.DefaultCaller)

	// trap SIGINT to trigger a shutdown.
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)

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

	// start the HTTP listener for metrics
	go func() {
		logger.Log("msg", fmt.Sprintf("listening HTTP (metrics & map) on %v", *httpPort))
		logger.Log("err", http.ListenAndServe(fmt.Sprintf(":%s", *httpPort), nil))
	}()

	// start many go routines with clients
	jobs := make([]*Client, *clients)
	ctx := context.Background()
	jobChan := make(chan int)
	jobCounter := 0
	for i := 0; i < *clients; i++ {
		jobs[i] = NewClient(strconv.Itoa(i), logger, *debug, *withTLS, *insecureSkipVerify)
		go jobs[i].Start(ctx, jobChan, *server, strconv.Itoa(i), i)
		jobCounter++
		// delay the clients creation by 100ms
		time.Sleep(100 * time.Millisecond)
		if jobCounter%10 == 0 {
			logger.Log("info", "job counter", "count", jobCounter)
		}
	}

	// wait for the clients to finish the work
	for {
		select {
		case wdone := <-jobChan:
			logger.Log("info", "job finished", "state", "OK", "ID", wdone)
			jobCounter--
		case <-signals:
			return
		case <-time.After(20 * time.Second):
			logger.Log("info", "jobCounter", "jobCounter", jobCounter)
			if jobCounter == 0 {
				logger.Log("info", "no more jobs")
				return
			}
		}
	}
}
