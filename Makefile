all: test

models:
	@flatc --go --gen-object-api ./models.fbs

test: models
	@goimports -w .
	@go test -timeout 10s -race -count 10 -cover -gcflags=all=-d=checkptr=0 -coverprofile=./journal.cover ./...

cover: test
	@go tool cover -html=./journal.cover
