lint:
	docker run --rm -v $(PWD):/app -w /app golangci/golangci-lint:v1.27.0 golangci-lint run --skip-files ".*\\.pb\\.go" --allow-parallel-runners ./...
	
vet:
	go vet ./...

tests:
	go test -v ./...

generate-proto:
	# go get -u google.golang.org/grpc
	# go get -u github.com/golang/protobuf/protoc-gen-go
	# PATH=$(PATH):`pwd`/../protoc/bin:$(GOPATH)/bin
	# export PATH=$PATH:`pwd`/../protoc/bin:$GOPATH/bin
	go generate