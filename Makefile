all: build

build:
	go build -o notes .

run: build
	./notes

fmt:
	gofmt -w .

lint:
	go vet ./..

clean:
	rm -rf notes