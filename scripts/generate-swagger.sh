#!/usr/bin/env sh
set -eu

cd "$(dirname "$0")/.."

if ! command -v swag >/dev/null 2>&1; then
  echo "swag is not installed. Install with:"
  echo "  go install github.com/swaggo/swag/cmd/swag@latest"
  exit 1
fi

swag init -g cmd/api/main.go -o docs
echo "Swagger written to docs/"
