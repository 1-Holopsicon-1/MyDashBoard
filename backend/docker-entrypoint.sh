#!/bin/sh
set -e
mkdir -p /app/data
chown -R app:app /app/data 2>/dev/null || true

DOCKER_GID=$(stat -c '%g' /var/run/docker.sock 2>/dev/null || echo "")
if [ -n "$DOCKER_GID" ]; then
	addgroup -g "$DOCKER_GID" docker 2>/dev/null || true
	addgroup app docker 2>/dev/null || true
fi

exec su-exec app "$@"