.PHONY: clean build

PWD=$(shell pwd)
VER?="1.1.0"

default: clean build

clean: 
	@rm -rf releases

build:
	@mkdir -p releases
	GO111MODULE=on CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o releases/docker .