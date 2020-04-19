all: proto build

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
	rm -f go.sum
	rm -f ./messaging/messaging.pb.go
	rm -f $$GOPATH/bin/ter-grpc
	rm -fr ./metrics

deploy:
	./scripts/deploy.sh -f ./machines.txt -u gaydamakha -p 5000 -n 2

stop:
	./scripts/stop.sh

.PHONY: certs proto build fmt clean metrics
