BINARY=dclean
INSTALL_DIR=$(shell go env GOPATH)/bin
LDFLAGS=-s -w
SUDOERS_SRC=packaging/dclean.sudoers.in
SUDOERS_DST=/etc/sudoers.d/dclean

# the parent directory has a go.work that does not list this module
export GOWORK=off

.PHONY: help build install uninstall run fmt vet lint check clean deps update sudoers unsudoers

help:
	@echo "make install    Compilar e instalar en $(INSTALL_DIR) + reglas sudo"
	@echo "make uninstall  Eliminar el binario instalado + reglas sudo"
	@echo "make sudoers    Instalar solo $(SUDOERS_DST)"
	@echo "make build      Compilar en el directorio actual"
	@echo "make run        Compilar y ejecutar"
	@echo "make check      fmt + vet + lint"
	@echo "make deps       go mod tidy"
	@echo "make update     Actualizar dependencias"
	@echo "make clean      Eliminar el binario local"

install: sudoers
	go build -ldflags="$(LDFLAGS)" -o $(INSTALL_DIR)/$(BINARY) ./cmd
	@echo "Instalado en $(INSTALL_DIR)/$(BINARY)"

# without this the system targets (snap, journal, apt, crash) fail with
# "sudo: se requiere una contraseña": the TUI runs sudo -n and cannot prompt
sudoers:
	@if [ -f $(SUDOERS_DST) ]; then echo "Reglas sudo ya instaladas en $(SUDOERS_DST)"; else \
		tmp=$$(mktemp); \
		sed "s|@USER@|$$(id -un)|" $(SUDOERS_SRC) > $$tmp; \
		visudo -c -f $$tmp >/dev/null || { rm -f $$tmp; exit 1; }; \
		sudo install -m 0440 -o root -g root $$tmp $(SUDOERS_DST); \
		rm -f $$tmp; \
		echo "Reglas sudo instaladas en $(SUDOERS_DST)"; \
	fi

unsudoers:
	sudo rm -f $(SUDOERS_DST)

uninstall: unsudoers
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
