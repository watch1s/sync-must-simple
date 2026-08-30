.PHONY: build build-windows clean

build:
	mkdir -p dist
	CGO_ENABLED=0 go build -ldflags "-s -w" -o dist/sync-must-simple-linux ./server

build-windows:
	mkdir -p dist
	CGO_ENABLED=0 GOOS=windows go build -ldflags "-H=windowsgui -s -w" -o dist/sync-must-simple.exe ./server

clean:
	rm -rf dist server/sync.db

