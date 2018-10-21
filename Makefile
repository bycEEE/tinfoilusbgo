PROJECT_NAME := tinfoilusbgo
LIBUSB_URL := https://github.com/libusb/libusb/releases/download/v1.0.22/libusb-1.0.22.tar.bz2
GO_VERSION := 1.11.1
LDFLAGS := -linkmode external -extldflags -static

build-windows:
	mkdir -p ./bin
	xgo -go $(GO_VERSION) --deps=$(LIBUSB_URL) --targets=windows-10.0/amd64 -ldflags="$(LDFLAGS)" -buildmode=exe -out ./bin/$(PROJECT_NAME) .

	# build with mingw-w64 installed locally
	# env GOOS="windows  CGO_ENABLED="1" GOARCH="amd64" CC="x86_64-w64-mingw32-gcc" go build -ldflags "-linkmode external -extldflags -static" -o "./bin/tinfoilusbgo_64.exe"

# build-osx:
# 	mkdir -p ./bin
# 	GOARCH="amd64" go build -o ./bin/$(PROJECT_NAME)-osx-$(shell sw_vers -productVersion)-amd64 .

# build-linux: # untested
# 	mkdir -p ./bin
# 	xgo -go $(GO_VERSION) --deps="$(LIBUSB_URL)" --targets=linux/amd64 -out ./bin/$(PROJECT_NAME) .

build-all: build-windows build-osx build-linux
.PHONY: build-all build-windows build-osx build-linux
