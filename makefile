build:
	go build -o bin/golang-json-api

run: build
	./bin/golang-json-api

runapp:
	./bin/golang-json-api

test:
	go test -v ./...