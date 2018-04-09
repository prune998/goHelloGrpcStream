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
	"io"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/grpc-ecosystem/go-grpc-prometheus"

	kitlog "github.com/go-kit/kit/log"
	"github.com/namsral/flag"
	pb "github.com/prune998/goHelloGrpcStream/helloworld/helloworld"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

var (
	debug   = flag.Bool("debug", false, "display debugs")
	server  = flag.String("server", "localhost:7788", "Greeter Server URL")
	name    = flag.String("name", "world", "name of the client (will be displayed in the server)")
	clients = flag.Int("clients", 1, "number of clients to simulate")
)

type Client struct {
	kitlog.Logger
	ID    string `json:"device_id"`
	debug bool
}

// NewClient creates a new client
func NewClient(id string, logger kitlog.Logger, debug bool) *Client {
	if debug {
		logger.Log("msg", "starting client "+id)
	}

	return &Client{
		Logger: logger,
		ID:     id,
		debug:  debug,
	}
}

// Start a new client
func (c Client) Start(ctx context.Context, jobChan chan<- int, server, name string, id int) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	conn, err := grpc.Dial(server,
		grpc.WithInsecure(),
		grpc.WithUnaryInterceptor(grpc_prometheus.UnaryClientInterceptor),
		grpc.WithStreamInterceptor(grpc_prometheus.StreamClientInterceptor),
	)
	if err != nil {
		c.Logger.Log("msg", "cant connect to server", "err", err, "ID", c.ID)
		jobChan <- id
		return
	}
	defer conn.Close()
	g := pb.NewGreeterClient(conn)

	// Contact the server and print out its response.
	r, err := g.SayHello(ctx, &pb.HelloRequest{Name: name})
	if err != nil {
		c.Logger.Log("msg", "could not greet server", "err", err, "ID", c.ID)
		jobChan <- id
		return
	}
	PromSayHelloReceivedCounter.Inc()
	if c.debug {
		c.Logger.Log("msg", "Received Greeting: "+r.Message, "ID", c.ID)
	}

	stream, err := g.SayHelloStream(context.Background())
	if err != nil {
		c.Logger.Log("msg", "could not SayHelloStream", "err", err, "ID", c.ID)
		jobChan <- id
		return
	}
	PromSayHelloStreamReceivedCounter.Inc()
	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			c.Logger.Log("msg", "got EOF from server", "err", err, "ID", c.ID)
			break
		}
		if err != nil {
			//log.Fatalf("%v.GetCustomers(_) = _, %v", c, err)
			c.Logger.Log("msg", "got error from server", "err", err, "ID", c.ID)
			break
		}
		if c.debug {
			c.Logger.Log("msg", msg.Message, "ID", c.ID)
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

	// start many go routines with clients
	jobs := make([]*Client, *clients)
	ctx := context.Background()
	jobChan := make(chan int)
	jobCounter := 0
	for i := 0; i < *clients; i++ {
		jobs[i] = NewClient(strconv.Itoa(i), logger, *debug)
		go jobs[i].Start(ctx, jobChan, *server, strconv.Itoa(i), i)
		jobCounter++
	}

	for {
		select {
		case wdone := <-jobChan:
			logger.Log("info", "job finished", "state", "OK", "ID", wdone)
			jobCounter--
		case <-signals:
			return
		case <-time.After(20 * time.Second):
			logger.Log("info", "jobCounter", "jobCounter", jobCounter)
		}
	}

	// go func(ctx context.Context) {
	// 	for i := 0; i < clients; i++ {
	// 		go d.startClient(ctx, i, clients, maxDelay, predictable)
	// 	}
	// 	for {
	// 		select {
	// 		case <-ctx.Done():
	// 			d.Log("msg", "calling off attack")
	// 			return
	// 		}
	// 	}
	// }(ctx)
	// Set up a connection to the server.
	// conn, err := grpc.Dial(*server, grpc.WithInsecure())
	// if err != nil {
	// 	logger.Log("msg", "cant connect to server", "err", err)
	// }
	// defer conn.Close()
	// c := pb.NewGreeterClient(conn)

	// // Contact the server and print out its response.
	// r, err := c.SayHello(context.Background(), &pb.HelloRequest{Name: *name})
	// if err != nil {
	// 	logger.Log("msg", "could not greet server", "err", err)
	// }
	// logger.Log("msg", "Received Greeting: "+r.Message)

	// request for the Stream
	// stream, err := c.SayHelloStream(context.Background())

	// for {
	// 	msg, err := stream.Recv()
	// 	if err == io.EOF {
	// 		logger.Log("msg", "got EOF from server", "err", err)
	// 		break
	// 	}
	// 	if err != nil {
	// 		//log.Fatalf("%v.GetCustomers(_) = _, %v", c, err)
	// 		logger.Log("msg", "got error from server", "err", err)
	// 		break
	// 	}
	// 	logger.Log("msg", msg.Message)
	// }
}
