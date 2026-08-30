.PHONY: deps build

deps:
	sudo apt-get update
	sudo apt-get install -y libgtk-3-dev libappindicator3-dev gcc

build:
	mkdir -p dist
	CGO_ENABLED=1 go build -o dist/sync-must-simple-linux -tags systray ./server
