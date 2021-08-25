.PHONY: clean build

PWD=$(shell pwd)
VER?="1.1.0"

default: clean build

clean: 
	@rm -rf releases

build:
	@mkdir -p releases
	@mkdir -p releases/linux/amd64
	@mkdir -p releases/linux/arm
	@mkdir -p releases/linux/arm64

# 	linux/amd64
	docker run -it --rm \
		-v $(PWD):/app \
		--env GO111MODULE=on \
		--env CGO_ENABLED=0 \
		--env GOOS=linux \
		--env GOARCH=amd64 \
		--workdir=/app \
		golang:1.16.5-alpine3.13 \
		go build -ldflags="-s -w" -o releases/linux/amd64/docker .
# 	linux/arm
	docker run -it --rm \
		-v $(PWD):/app \
		--env GO111MODULE=on \
		--env CGO_ENABLED=0 \
		--env GOOS=linux \
		--env GOARCH=arm \
		--workdir=/app \
		golang:1.16.5-alpine3.13 \
		go build -ldflags="-s -w" -o releases/linux/arm/docker .
# 	linux/arm64
	docker run -it --rm \
		-v $(PWD):/app \
		--env GO111MODULE=on \
		--env CGO_ENABLED=0 \
		--env GOOS=linux \
		--env GOARCH=arm \
		--workdir=/app \
		golang:1.16.5-alpine3.13 \
		go build -ldflags="-s -w" -o releases/linux/arm/docker .
# 	windows/amd64
	docker run -it --rm \
		-v $(PWD):/app \
		--env GO111MODULE=on \
		--env CGO_ENABLED=0 \
		--env GOOS=windows \
		--env GOARCH=amd64 \
		--workdir=/app \
		golang:1.16.5-alpine3.13 \
		go build -ldflags="-s -w" -o releases/windows/amd64/docker.exe .

	docker build --force-rm -t xuanloc0511/drone-plugin-docker:$(VER) -f Dockerfile $(PWD)
	docker tag xuanloc0511/drone-plugin-docker:$(VER) xuanloc0511/drone-plugin-docker:latest
	make clean