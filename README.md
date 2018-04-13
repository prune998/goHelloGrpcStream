# goHelloGrpcStream
a test Hello world application using GRPC streaming in GO

## Usage

### Proto file
use `protoc` if you need to re-generate the proto files.
Refer to https://github.com/grpc/grpc-go/tree/master/examples if you need help.

```
cd helloworld
protoc -I helloworld/ helloworld/helloworld.proto --go_out=plugins=grpc:helloworld
```

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
 - ~~request a `hello world` message and display the return from the server~~
 - request a `hello world` streaming message and display each message until the server stops sending

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
Use the docker file to build an image embedding both client and server code.
Best is to use the makefile : 
```
make docker
make docker-push
```
or by hand :

```
docker build -t prune/gohellogrpcstream:latest .
docker push prune/gohellogrpcstream:latest
```

## Kubernetes

Deploys a server + service that listen on port 7788 along with a load-tester client

```
    kubectl -n dev apply -f kubernetes/deployment-autoinject-istio.yml
```

# Load Test

## client
Use the `loadtest_client` which is simulating any number of clients in parallel.
The `loadtest_client` make one TCP connexion per client which then open a `gRPC HTTP/2 Stream`. Starting 100 clients should show you 100 POST requests on `/helloworld.Greeter/SayHelloStream`. 
When stopping the client, the 100 streams will be closed, showing 100 `/helloworld.Greeter/SayHelloStream` POSTs.

You can then scale the number of connexions by either :
- editing the number of `clients` from the ENV
- scaling to more pods

## ingress
You can test the client/server inside K8s but from an external point of view using an Ingress.
Update the `kubernetes/ingress.yml` file with a public DNS name you own then deploy using : 
```
kubectl -n dev apply -f kubernetes/ingress.yml
```

Then update the `greeter-client` or `loadtest-client` to use the external endpoint : 
```
...
      containers:
      - name: loadtest-client
        image: "prune/gohellogrpcstream:latest"      
        ports:
        - containerPort: 7788
          name: loadtest-client
        command: 
          - "/root/loadtest_client"
        env:
          - name: "CLIENTS"
            value: "10"
          - name: "SERVER"
            value: "your-external-dns-name:80"
...
```