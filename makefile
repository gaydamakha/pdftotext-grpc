all: certs proto build

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

proto:
	protoc \
		./messaging/messaging.proto \
		--go_out=plugins=grpc:.

build:
	go install .

fmt:
	go fmt
	cd ./client && go fmt
	cd ./server && go fmt
	cd ./cmd && go fmt

clean:
	rm go.sum
	rm ./messaging/messaging.pb.go
	rm $$GOPATH/bin/ter-grpc

serve-tls:
	$$GOPATH/bin/ter-grpc serve \
        --key ./certs/localhost.key \
        --certificate ./certs/localhost.cert

upload-tls:
	$$GOPATH/bin/ter-grpc upload \
        --root-certificate ./certs/localhost.cert \
        --file $(file)

.PHONY: certs proto build fmt clean