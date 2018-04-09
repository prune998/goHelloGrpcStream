package main

import "github.com/prometheus/client_golang/prometheus"

var (
	PromSayHelloReceivedCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "greeter_server_SayHello_received_counter",
		Help: "Unary SayHello requests received",
	})

	PromSayHelloStreamReceivedCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "greeter_server_SayHelloStream_received_counter",
		Help: "SayHelloStream requests received",
	})
)

func init() {
	prometheus.MustRegister(PromSayHelloReceivedCounter)
	prometheus.MustRegister(PromSayHelloStreamReceivedCounter)
}
