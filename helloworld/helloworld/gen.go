//go:generate protoc  --proto_path=../..:.:$GOPATH/src --go_out==plugins=grpc:. helloworld.proto

package helloworld
