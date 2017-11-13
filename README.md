# goHelloGrpcStream
a test Hello world application using GRPC streaming in GO

## Usage

### Server
The server opens a TCP socket and wait for GRPC messages to come in

    ```
    cd helloworld/greeter_server/
    go build
    ./greeter_server

2017/11/13 10:35:54 Listening on tcp://localhost: :7788
```

### Client
The client connect to the server on the provided `host:port` and :
 - request a `hello world` message and display the return from the server
 - request a `hello wold` streaming message and display each message until the server stops sending

    ```
    cd helloworld/greeter_client/
    go build
    ./greeter_client localhost:7788

2017/11/13 10:26:04 Greeting: Hello world:7788
2017/11/13 10:26:09 Hello Streamworld:7788
2017/11/13 10:26:14 Hello Streamworld:7788
2017/11/13 10:26:19 Hello Streamworld:7788
```

## Docker
Use the docker file to build an image embedding both client and server code

```
docker build -t prune/gohellogrpcstream:latest .
docker push prune/gohellogrpcstream:latest
```

## Kubernetes

### server.yml
    Deploys a server + service thatl isten on port 7788

    ```
    kubectl -n dev apply -f server.yml
    ```

### client.yml
    Deploys a client that connect to the server on `greeter_server:7788`

    ```
    kubectl -n dev apply -f client.yml
    ```