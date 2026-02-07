.PHONY: build clean install test plugin-desktop plugin-clean web

VERSION ?= 1.0.0

build:
	go build -o loom

install: build
	cp loom $(HOME)/.local/bin/

clean:
	rm -f loom

test:
	go test ./...

run: build
	./loom

# Run the web dashboard server
web: build
	./loom -addr :8080 -web-addr :3000

# --- Plugin targets ---

plugin-desktop:
	@mkdir -p dist desktop-extension/server
	GOOS=darwin GOARCH=arm64 go build -o desktop-extension/server/loom && \
		cd desktop-extension && mcpb pack && mv *.mcpb ../dist/loom-darwin-arm64-$(VERSION).mcpb
	GOOS=darwin GOARCH=amd64 go build -o desktop-extension/server/loom && \
		cd desktop-extension && mcpb pack && mv *.mcpb ../dist/loom-darwin-amd64-$(VERSION).mcpb
	GOOS=linux GOARCH=amd64 go build -o desktop-extension/server/loom && \
		cd desktop-extension && mcpb pack && mv *.mcpb ../dist/loom-linux-amd64-$(VERSION).mcpb
	rm -rf desktop-extension/server

plugin-clean:
	rm -rf dist/ desktop-extension/server/
