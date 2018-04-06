FROM golang:1.9 as builder

LABEL vendor="Prune - prune@lecentre.net" \
      content="helloworld"

COPY helloworld /go/src/github.com/prune998/goHelloGrpcStream/helloworld
WORKDIR /go/src/github.com/prune998/goHelloGrpcStream/helloworld

RUN    { go get github.com/golang/protobuf || true; } && \
  go get  golang.org/x/net/context && \
  go get  google.golang.org/grpc
RUN    cd greeter_server && CGO_ENABLED=0 GOOS=linux go build -v
RUN    cd greeter_client && CGO_ENABLED=0 GOOS=linux go build -v
RUN    cd loadtest_client && CGO_ENABLED=0 GOOS=linux go build -v

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=0 /go/src/github.com/prune998/goHelloGrpcStream/helloworld/greeter_server/greeter_server .
COPY --from=0 /go/src/github.com/prune998/goHelloGrpcStream/helloworld/greeter_client/greeter_client .
COPY --from=0 /go/src/github.com/prune998/goHelloGrpcStream/helloworld/loadtest_client/loadtest_client .

# GRPC port
EXPOSE 7788

