.PHONY : preproc
preproc: clean fmt test

.PHONY : fmt
fmt:
	go fmt ./...
	goimports -v -w .

.PHONY : test
test:
	go test ./... -race -coverprofile=cover.out -covermode=atomic

.PHONY : clean
clean:

.PHONY : check-coverage
check-coverage:
	go tool cover -html cover.out

