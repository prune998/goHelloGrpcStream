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

The server does not support HTTPS yet. You have to use a middleware (Istio/Envoy, Traefik, Nginx...) to handle the TLS termination and send plain HTTP/2 to the server

### Client
The client connect to the server on the provided `host:port` and : 
 - ~~request a `hello world` message and display the return from the server~~
 - request a `hello world` streaming message and display each message until the server stops sending

```
cd helloworld/greeter_client/
go build
./greeter_client -debug -server=localhost:7788 -unary

2017/11/13 10:26:04 Greeting: Hello world:7788
2017/11/13 10:26:09 Hello Streamworld:7788
2017/11/13 10:26:14 Hello Streamworld:7788
2017/11/13 10:26:19 Hello Streamworld:7788
```

The client support two modes :
 - unary : will send a single HTTP/2 request
   This is usefull to test the low level connectivity
 - stream : will send a HTTP/2 di-directional Stream request and keep the stream opened
  This is usefull to test the longevity of the connection and the number of possible parallel connections

The client support TLS, see `-h` for options

### Loadtest
The loadtest application opens one HTTP/2 streaming connection per `-clients` and maintain it for `-cnxDelay`

The loadtest support TLS, see `-h` for options

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

The `client` pod will not do anything so Istio have the time to settle. 
Start the `loadtest_client` using : 

```
kubectl exec -ti $(kubectl get pods --selector=app=loadtest-client -o jsonpath='{.items..metadata.name}')  /root/loadtest_client

```

# Load Test

## client
Use the `loadtest_client` which is simulating any number of clients in parallel.
The `loadtest_client` make one TCP connexion per client which then open a `gRPC HTTP/2 Stream`. Starting 100 clients should show you 100 POST requests on `/helloworld.Greeter/SayHelloStream`. 
When stopping the client, the 100 streams will be closed, showing 100 `/helloworld.Greeter/SayHelloStream` POSTs.

You can then scale the number of connexions by either :
- editing the number of `clients` from the ENV
- scaling to more pods


...
```

# Istio
This tool was primilary made to test Istio Service Mesh deployed in Kubernetes.
in the Kubernetes folder you will find the needed files to deploy the pods and create the Istio config.

## Using Ingress (Istio < 0.8.0)
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
            value: "greet.dev.mydomain.com:80"

You may use the `Destination Policy` and `Destination Rule` files to better configure the limits of the Istio Ingress. 
Note that Istio < 1.x have some issues with the Max Concurent Streams values. 

## Using Istio > 0.8.0 (including 1.0.0+)
In 0.8.0 and 1.x, Istio changed his setup and no longer use the Kubernetes Ingress. 
Instead, Istio is using a `LoadBalancer Service` to forward all the traffic to `Istio IngressGateway`.
We here consider you have a working Istio Service Mesh. Refer to Istio docs otherwise.

On the Istio side, you now need to create a `Gateway` that will allow incoming traffic to the Mesh and a `VirtualService` that will forward this traffic to a Kubernetes Service.

Deploy the `gateway.yml` then the `virtualservice.yml`. 
You should now be able to use the `greeter_client` or the `loadtest_client` by pointing it to `hello.test.domain.ca:80` or `hello.test.domain.ca:443`.
Don't forger to replace all the demo domain names by your own, and propagate the DNS changes :)
