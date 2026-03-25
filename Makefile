all: build

build:
	go build -o bin/larry .

run: build
	./bin/larry

fmt:
	gofmt -w .

lint:
	go vet ./..

clean:
	rm -rf bin/notes