#!/usr/bin/env bash
set -euo pipefail

PROJECT_ROOT="$(cd "$(dirname "$0")/.." && pwd)"

echo "==> Building Go backend..."
cd "$PROJECT_ROOT/backend"
go build -o "$PROJECT_ROOT/backend/server" ./cmd/server

echo "==> Building SvelteKit frontend..."
cd "$PROJECT_ROOT/frontend"
yarn build

echo "==> Done. Artifacts:"
echo "    Backend: backend/server"
echo "    Frontend: frontend/build/"