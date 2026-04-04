#!/bin/bash
set -e

cd portfolio

# Initialize Go module if not exists
if [ ! -f go.mod ]; then
  go mod init portfolio
fi

# Replace nexgo with local path
go mod edit -replace github.com/salmanfaris22/nexgo=../

# Require nexgo module
go mod edit -require github.com/salmanfaris22/nexgo@v0.0.0

# Download dependencies
go mod tidy

# Build using nexgo CLI
go run ../cmd/nexgo/main.go build .

echo "Build complete! Output in .nexgo/out/"
