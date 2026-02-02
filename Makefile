.PHONY: build run debug test clean

build:
	go build -o lazyfirewall ./cmd/lazyfirewall

run: build
	sudo ./lazyfirewall

debug: build
	LAZYFIREWALL_DEBUG=1 sudo ./lazyfirewall

test:
	go test ./...

clean:
	rm -f lazyfirewall
