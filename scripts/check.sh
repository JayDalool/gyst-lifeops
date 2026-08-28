#!/usr/bin/env bash
# Verification for gyst-lifeops.
#
# There is no Go toolchain installed on this host - Go exists only inside
# Docker (see Dockerfile, which builds from golang:1.25-bookworm). This script
# runs gofmt, go vet, and go test in a throwaway container against the working
# tree, and caches the module downloads in a named volume so repeat runs are
# fast.
#
# Usage: ./scripts/check.sh          # fmt + vet + test
#        ./scripts/check.sh test     # test only
set -euo pipefail

cd "$(dirname "$0")/.."

MODE="${1:-all}"
IMAGE="golang:1.25-bookworm"
CACHE="gyst-gomodcache"

docker volume create "$CACHE" >/dev/null

run() {
  docker run --rm \
    -v "$PWD":/app \
    -v "$CACHE":/go/pkg/mod \
    -w /app \
    "$IMAGE" sh -c "$1"
}

case "$MODE" in
  test) run 'go test ./...' ;;
  *)
    echo "==> gofmt"
    unformatted=$(run "gofmt -l .")
    if [ -n "$unformatted" ]; then
      echo "unformatted files:"; echo "$unformatted"; exit 1
    fi
    echo "==> go vet"
    run 'go vet ./...'
    echo "==> go test"
    run 'go test ./...'
    ;;
esac

echo "==> all checks passed"
