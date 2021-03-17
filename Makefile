.PHONY: build run

build:
	GOOS=$(os) GOARCH=$(arch) go build -o assets/bot .  

build-default:
	make build os=linux arch=amd64

build-arm:
	make build os=linux arch=arm64

run:build-default
	cd assets; ./bot; rm ./bot
