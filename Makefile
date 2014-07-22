DEPS = $(shell go list -f '{{range .TestImports}}{{.}} {{end}}' ./...)

all: deps format
	@mkdir -p bin/
	@bash --norc -i ./scripts/build.sh

deps:
	@echo "--> Installing build dependencies"
	@go get -d -v ./...
	@echo $(DEPS) | xargs -n1 go get -d

format:
	@echo "--> Running go fmt"
	@gofmt -w .

test: deps
	go test ./...

release:
	@bash --norc -i ./scripts/release.sh

clean:
	rm -rf crosby/pkg
	rm -rf bin

.PHONY: all deps test format
