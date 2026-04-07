#!/bin/sh
set -e

# Ensure data directories exist (handles empty bind mounts)
mkdir -p /data/notes /data/uploads

exec /app/server "$@"
