all: build

build:
	go build -o bin/larry .

run: build
	./bin/larry

dev: build
	npm --prefix scripts run watch > /tmp/tsc-watch.log 2>&1 & ./bin/larry; kill %1

fmt:
	gofmt -w .

lint:
	go vet ./..

clean:
	rm -rf bin/notes