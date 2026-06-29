BINARY=scripty
FIXMIC_BINARY=fix-mic

.PHONY: all build build-fixmic install uninstall clean

all: build build-fixmic

build:
	go build -o $(BINARY) ./cmd/scripty/

build-fixmic:
	go build -o $(FIXMIC_BINARY) ./cmd/fix-mic/

install: build
	./$(BINARY) install

uninstall:
	rm -f ~/.local/bin/$(BINARY)
	@echo "Removed ~/.local/bin/$(BINARY)"

clean:
	rm -f $(BINARY) $(FIXMIC_BINARY)
