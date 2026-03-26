all: build

build:
	go build -o bin/larry ./main.go

run: build
	./bin/larry

dev: build
	rm -rf scripts/dist && mkdir -p scripts/dist && npm --prefix scripts run watch > /tmp/tsc-watch.log 2>&1 & ./bin/larry; kill %1

fmt:
	gofmt -w .

lint:
	go vet ./..

clean:
	rm -rf bin/notes