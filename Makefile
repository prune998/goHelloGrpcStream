version := $(shell git describe  --always --tags --long --abbrev=8)

buildtime := $(shell date -u +%Y%m%d.%H%M%S)

all: cmds

lint:
	@gometalinter --disable-all --enable=vet --enable=vetshadow  --enable=structcheck \
	    --enable=deadcode --enable=gotype --enable=goconst --enable=golint --enable=varcheck \
	     --enable=unconvert --enable=staticcheck --enable=gas --enable=dupl --enable=ineffassign \
	     --enable=gocyclo --cyclo-over=20 --vendor ./...

docker:
	docker build -t prune/gohellogrpcstream:$(version) .

docker-push:
	docker push prune/gohellogrpcstream:$(version)
	docker tag prune/gohellogrpcstream:$(version) prune/gohellogrpcstream:latest
	docker push prune/gohellogrpcstream:latest
protos: helloworld/helloworld.pb.go

helloworld/helloworld.pb.go:
	# cd helloworld/helloworld && go generate
	cd helloworld && protoc -I helloworld/ helloworld/helloworld.proto --go_out=plugins=grpc:helloworld

greeter_client: test
	cd helloworld/greeter_client && CGO_ENABLED=0 GOOS=linux go build -v -ldflags "-X main.version=$(version)-$(buildtime)" 

greeter_server: test
	cd helloworld/greeter_server && CGO_ENABLED=0 GOOS=linux go build -v -ldflags "-X main.version=$(version)-$(buildtime)" 

loadtest_client: test
	cd helloworld/loadtest_client && CGO_ENABLED=0 GOOS=linux go build -v -ldflags "-X main.version=$(version)-$(buildtime)" 

cmds: greeter_client greeter_server loadtest_client

test:
	go test ./...

clean:
	rm -f ./helloworld/greeter_server/greeter_server \
	  ./helloworld/greeter_client/greeter_client \
	  ./helloworld/loadtest_client/loadtest_client
