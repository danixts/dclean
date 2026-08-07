BINARY=dclean
INSTALL_DIR=$(shell go env GOPATH)/bin
LDFLAGS=-s -w

# the parent directory has a go.work that does not list this module
export GOWORK=off

.PHONY: help build install uninstall run fmt vet lint check clean deps update

help:
	@echo "make install    Compilar e instalar en $(INSTALL_DIR)"
	@echo "make uninstall  Eliminar el binario instalado"
	@echo "make build      Compilar en el directorio actual"
	@echo "make run        Compilar y ejecutar"
	@echo "make check      fmt + vet + lint"
	@echo "make deps       go mod tidy"
	@echo "make update     Actualizar dependencias"
	@echo "make clean      Eliminar el binario local"

install:
	go build -ldflags="$(LDFLAGS)" -o $(INSTALL_DIR)/$(BINARY) ./cmd
	@echo "Instalado en $(INSTALL_DIR)/$(BINARY)"

uninstall:
	rm -f $(INSTALL_DIR)/$(BINARY)

build:
	go build -ldflags="$(LDFLAGS)" -o $(BINARY) ./cmd

run: build
	./$(BINARY)

fmt:
	gofmt -w .

vet:
	go vet ./...

lint:
	golangci-lint run ./...

check: fmt vet lint

clean:
	rm -f $(BINARY)

deps:
	go mod tidy

update:
	go get -u ./...
	go mod tidy
