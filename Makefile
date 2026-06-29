BINARY=fix-mic

.PHONY: all build clean

all: build

build:
	go build -o $(BINARY) .

clean:
	rm -f $(BINARY)
