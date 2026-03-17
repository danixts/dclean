BINARY=dclean
INSTALL_DIR=$(shell go env GOPATH)/bin

.PHONY: build install clean run deps update

build:
	go build -o $(BINARY) ./cmd

install: build
	cp $(BINARY) $(INSTALL_DIR)/$(BINARY)

clean:
	rm -f $(BINARY)

run: build
	./$(BINARY)

deps:
	go mod tidy

update:
	go get -u ./...
	go mod tidy
