package main

// Build metadata. Defaults keep `go build ./...` and `go vet ./...` working on a
// clean checkout; release builds inject real values with:
//
//	-ldflags "-X main.version=1.2.3 -X main.commitID=$(git rev-parse HEAD)"
var (
	version  = "dev"
	commitID = "unknown"
)
