include .env
export

debug:
	@echo "MIGRATIONS_PATH=$(MIGRATIONS_PATH)"
	@echo "realpath MIGRATIONS_PATH=$(realpath $(MIGRATIONS_PATH))"


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

.PHONY : up
up:
	docker-compose up -d

.PHONY: down
down:
	docker-compose down -v


.PHONY: migrate
migrate:
	docker run --rm \
		-v $(realpath $(MIGRATIONS_PATH)):/migrations \
		--network=gophermart-network \
		migrate/migrate:v4.18.3 \
			-path=/migrations \
			-database postgres://gophermart:gophermart@gophermart-database:5432/gophermart?sslmode=disable \
			up
