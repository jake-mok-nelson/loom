.PHONY: build clean install test

build:
	go build -o loom

install: build
	cp loom $(HOME)/.local/bin/

clean:
	rm -f loom
	rm -rf $(HOME)/.loom/loom.db

test:
	go test ./...

run: build
	./loom
