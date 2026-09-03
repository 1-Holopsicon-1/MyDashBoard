#!/bin/sh
set -e
mkdir -p /app/data
chown -R app:app /app/data 2>/dev/null || true
exec su-exec app "$@"