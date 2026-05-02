.PHONY: default vet test format site good

default:
	@echo "target required"
	@exit 1

vet:
	go vet ./...

test:
	go test ./...

format:
	gofmt -w $$(git ls-files '*.go')

site:
	rm -rf _site
	mkdir -p _site
	go run ./cmd/mdori -o _site/index.html README.md
	go run ./cmd/mdori -o _site/examples.html site/examples.md
	cp -R site/assets _site/assets

good: format vet test
