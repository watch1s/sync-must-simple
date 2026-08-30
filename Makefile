.PHONY: build build-windows clean

build:
	mkdir -p dist
	cd server && CGO_ENABLED=0 go build -buildvcs=false -ldflags "-s -w" -o ../dist/sync-must-simple-linux .

build-windows:
	mkdir -p dist
	cd server && CGO_ENABLED=0 GOOS=windows go build -buildvcs=false -ldflags "-H=windowsgui -s -w" -o ../dist/sync-must-simple.exe .

clean:
	rm -rf dist server/sync.db

