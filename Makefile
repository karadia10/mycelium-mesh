SHELL := /bin/bash

.PHONY: all build workloads spore publish run test clean

all: build

build:
	go build ./...

workloads:
	mkdir -p bin
	go build -o bin/billing ./cmd/workload-billing
	go build -o bin/frontend ./cmd/workload-frontend

spore: workloads
	@echo "Run mesh build with your manifest, e.g.:"
	@echo "  go run ./cmd/mesh build -manifest ./examples/billing.json -binary ./bin/billing -out ./out"

publish:
	@echo "Run mesh publish, e.g.:"
	@echo "  go run ./cmd/mesh publish -spore $$(ls out/*.spore) -repo ./repo"

run:
	@echo "Run mesh with digest:"
	@echo "  go run ./cmd/mesh run -repo ./repo -digest <DIGEST> -app billing -instances 2 -edge :8080 -nodes 3"

test:
	go test ./... -v

clean:
	rm -rf bin out repo run
