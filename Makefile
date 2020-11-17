all: test

models:
	@flatc --go --gen-object-api ./models.fbs

test: models
	@goimports -w .
	@go test -timeout 30s -race -count 30 -cover -gcflags=all=-d=checkptr=0 -coverprofile=./journal.cover .
	@go test -timeout 30s -race -count 30 -cover -gcflags=all=-d=checkptr=0 -coverprofile=./crash.cover ./crash

cover: test
	@go tool cover -html=./crash.cover
	@go tool cover -html=./journal.cover
