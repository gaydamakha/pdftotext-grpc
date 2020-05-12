all: build

certs:
	mkdir -p ./certs
	openssl genrsa \
		-out ./certs/localhost.key \
		2048
	openssl req \
		-new -x509 \
		-key ./certs/localhost.key \
		-out ./certs/localhost.cert \
		-days 3650 \
		-subj /CN=localhost
build:
	go install github.com/golang/protobuf/protoc-gen-go
	protoc ./messaging/messaging.proto --go_out=plugins=grpc:. --go_opt=paths=source_relative 
	go install .

fmt:
	go fmt
	cd ./client && go fmt
	cd ./server && go fmt
	cd ./cmd && go fmt

clean:
	rm -f go.sum
	rm -f ./messaging/messaging.pb.go
	rm -f $$GOPATH/bin/ter-grpc
	rm -fr ./results/
	rm -fr ./txt/
	rm -fr ./certs/

metrics: build certs
	./scripts/collect_metrics.sh

.PHONY: certs proto build fmt clean metrics
