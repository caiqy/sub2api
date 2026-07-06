#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../../../.." && pwd -P)"

cd "$ROOT/backend"
go test ./... -run TestDoesNotExist -count=0

cd "$ROOT/frontend"
pnpm typecheck
