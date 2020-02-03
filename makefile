install:
	go install -i -v

fmt:
	go fmt
	cd ./client && go fmt
	cd ./server && go fmt

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

grpc:
	protoc \
		./pdftotext/pdftotext.proto \
		--gogofaster_out=plugins=grpc:.

.PHONY: fmt install grpc certs