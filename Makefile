.PHONY: go-deps
go-deps:
	go mod tidy
	go mod vendor
	go mod download

.PHONY: build
build:
	docker build -f Dockerfile.build -t builder .
	docker run --rm -it -v $$(pwd):/host --entrypoint cp builder /usr/local/bin/arpwatcher /host/arpwatcher.linux

