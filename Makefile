.PHONY: test build lint

test:
	go test ./...
	python3 -m py_compile loadgen/*.py

build:
	go build -o bin/sysmon ./cmd/sysmon/

lint:
	go vet ./...
	shellcheck --severity=warning kickstart.sh
