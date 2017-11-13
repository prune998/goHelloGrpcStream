FROM golang:1.9 as builder

LABEL vendor="Prune - prune@lecentre.net" \
      content="helloworld"

COPY helloworld /go/src/github.com/prune998/goHelloGrpcStream/helloworld
WORKDIR /go/src/github.com/prune998/goHelloGrpcStream/helloworld
RUN cd greeter_server && \
    go build && \
    cd ../greeter_client && \
    go build

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=0 /go/src/github.com/prune998/goHelloGrpcStream/helloworld/greeter_server/greeter_server .
COPY --from=0 /go/src/github.com/prune998/goHelloGrpcStream/helloworld/greeter_client/greeter_client .

# GRPC port
EXPOSE 7788

