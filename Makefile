# Copyright 2019 SolarWinds Worldwide, LLC.
# SPDX-License-Identifier: Apache-2.0

lint:
	docker run --rm -v $(PWD):/app -w /app golangci/golangci-lint:v1.27.0 golangci-lint run --allow-parallel-runners --timeout 3m ./...
	
vet:
	go vet ./...

tests:
	go test -v ./...

build:
	go build -o bin/rkubelog

docker:
	DOCKER_BUILDKIT=1 docker build -t quay.io/solarwinds/rkubelog .