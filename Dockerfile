FROM golang:1.13-alpine as builder

LABEL vendor="Prune - prune@lecentre.net" \
      content="helloworld"

ARG VERSION="0.1"
ARG BUILDTIME="20180411"

COPY . /go/src/github.com/prune998/goHelloGrpcStream
WORKDIR /go/src/github.com/prune998/goHelloGrpcStream/helloworld

RUN    cd    greeter_server  && CGO_ENABLED=0 GOOS=linux go build -v -mod=vendor -ldflags "-X main.version=${VERSION}-${BUILDTIME}" && \
       cd ../greeter_client  && CGO_ENABLED=0 GOOS=linux go build -v -mod=vendor -ldflags "-X main.version=${VERSION}-${BUILDTIME}" && \
       cd ../loadtest_client && CGO_ENABLED=0 GOOS=linux go build -v -mod=vendor -ldflags "-X main.version=${VERSION}-${BUILDTIME}"


FROM alpine:latest
RUN apk --no-cache add ca-certificates curl
WORKDIR /root/
COPY --from=0 /go/src/github.com/prune998/goHelloGrpcStream/helloworld/greeter_server/greeter_server .
COPY --from=0 /go/src/github.com/prune998/goHelloGrpcStream/helloworld/greeter_client/greeter_client .
COPY --from=0 /go/src/github.com/prune998/goHelloGrpcStream/helloworld/loadtest_client/loadtest_client .

# GRPC port
EXPOSE 7788

