all: certs build install

install:
	go install -v

fmt:
	go fmt
	cd ./client && go fmt
	cd ./server && go fmt
	cd ./cmd && go fmt

certs:
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
	go get -u gitlab.com/gaydamakha/ter-grpc
	go build

proto:
	protoc \
		./messaging/messaging.proto \
		--go_out=plugins=grpc:.

.PHONY: fmt install proto build certs