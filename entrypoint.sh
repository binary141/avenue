#!/bin/sh
set -e

PUID="${PUID:-1000}"
PGID="${PGID:-1000}"

# Make sure the uploads directory exists and is writable
mkdir -p /avenue/uploads
chown -R app:app /avenue/uploads

# Drop to the non-root user and run the app
exec su-exec app "$@"

