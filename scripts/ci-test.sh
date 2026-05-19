#!/usr/bin/env bash
set -euo pipefail

go test -timeout 20m \
  ./client \
  ./pkg/... \
  ./fingers \
  ./gogo \
  ./neutron \
  ./spray \
  ./zombie \
  ./examples \
  ./examples/cases/...

if [ -d cmd ]; then
  go test -timeout 20m ./cmd/...
  go build ./cmd/...
else
  echo "No cmd directory, skipping cmd package checks."
fi
