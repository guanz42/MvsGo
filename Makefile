.PHONY: all build run clean

CGO_ENABLED=1
INCLUDE_DIRS=/opt/MVS/include
LIBRARIES=/opt/MVS/lib
BINARY_NAME=MvsDemo

all: build run

build:
	CGO_ENABLED=$(CGO_ENABLED) GOOS=linux GOARCH=amd64 CGO_CFLAGS=-I$(INCLUDE_DIRS) CGO_LDFLAGS="-L$(LIBRARIES)/64 -Wl,-rpath=$(LIBRARIES)/64 -lMvCameraControl" go build -ldflags "-s -w" -o build/$(BINARY_NAME) main.go

run: build
	./build/$(BINARY_NAME)

clean:
	rm -r build/