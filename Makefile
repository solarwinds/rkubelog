setup-tools:
	GO111MODULE=off go get -u github.com/mgechev/revive;
	GO111MODULE=off go get -u github.com/kisielk/errcheck;
	GO111MODULE=off go get -u honnef.co/go/tools/cmd/staticcheck;
	GO111MODULE=off go get -u github.com/securego/gosec/cmd/gosec

lint:
	$(GOPATH)/bin/revive -config revive.toml

error_check:
	$(GOPATH)/bin/errcheck ./...

static_check:
	$(GOPATH)/bin/staticcheck -checks all ./...

vet:
	go vet ./...

sec_check:
	$(GOPATH)/bin/gosec ./...

tests:
	go test -v ./...

all_checks: vet lint error_check sec_check static_check

build:
	go build -o bin/cabbage

docker:
	docker build -t quay.io/solarwinds/cabbage .