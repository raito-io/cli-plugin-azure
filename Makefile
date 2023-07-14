GO := go

generate:
	go generate ./...

build: generate
       export GOPRIVATE=github.com/raito-io/cli-plugin-azure-ad
       go build ./...

unit-test:
	go test -mod=readonly -coverpkg=./... -covermode=atomic -coverprofile=unit-test-coverage.txt ./...

lint:
	golangci-lint run ./...
	go fmt ./...

test:
	go test -mod=readonly -tags=integration -race -coverpkg=./... -covermode=atomic -coverprofile=coverage.out.tmp ./...
	cat coverage.out.tmp | grep -v "/mock_" > coverage.txt #IGNORE MOCKS
	go tool cover -html=coverage.txt -o coverage.html
