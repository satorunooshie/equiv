.PHONY: test race vet fuzz bench bench-external examples verify

GO_CACHE ?= $${TMPDIR:-/tmp}/equiv-gocache
GO_MOD_CACHE ?= $${TMPDIR:-/tmp}/equiv-gomodcache

test:
	GOCACHE=$(GO_CACHE) go test ./...

race:
	GOCACHE=$(GO_CACHE) go test -race ./...

vet:
	GOCACHE=$(GO_CACHE) go vet ./...

fuzz:
	GOCACHE=$(GO_CACHE) go test . -run '^$$' -fuzz FuzzMapBytes -fuzztime=3s

bench:
	GOCACHE=$(GO_CACHE) go test -run '^$$' -bench . -benchtime=1x ./...

bench-external:
	cd bench && GOCACHE=$(GO_CACHE) GOMODCACHE=$(GO_MOD_CACHE) go test ./...
	cd bench && GOCACHE=$(GO_CACHE) GOMODCACHE=$(GO_MOD_CACHE) go test -run '^$$' -bench . -benchmem -benchtime=1x ./...

examples:
	@for dir in examples/*/; do GOCACHE=$(GO_CACHE) go run ./$$dir; done

verify: test race vet fuzz bench bench-external
