.PHONY: default vet test format good

default:
	@echo "target required"
	@exit 1

vet:
	go vet ./...

test:
	go test ./...

format:
	gofmt -w $$(git ls-files '*.go')

good: format vet test
