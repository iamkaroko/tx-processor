#!/bin/bash
set -e

mkdir -p \
  cmd/tx-processor/cli \
  internal/cache/redis \
  internal/config \
  internal/db/migrations \
  internal/handler \
  internal/middleware \
  internal/models \
  internal/processor \
  internal/repository \
  internal/server \
  internal/service \
  scripts

# cmd
cat > cmd/tx-processor/main.go <<EOF
package main
EOF

cat > cmd/tx-processor/cli/main.go <<EOF
package main
EOF

# internal packages
cat > internal/cache/cache.go <<EOF
package cache
EOF

cat > internal/cache/redis/cache.go <<EOF
package redis
EOF

cat > internal/config/config.go <<EOF
package config
EOF

cat > internal/db/db.go <<EOF
package db
EOF

cat > internal/handler/handler.go <<EOF
package handler
EOF

cat > internal/handler/analytics.go <<EOF
package handler
EOF

cat > internal/middleware/middleware.go <<EOF
package middleware
EOF

cat > internal/models/models.go <<EOF
package models
EOF

cat > internal/processor/processor.go <<EOF
package processor
EOF

cat > internal/repository/repository.go <<EOF
package repository
EOF

cat > internal/server/server.go <<EOF
package server
EOF

cat > internal/service/analytics.go <<EOF
package service
EOF

cat > scripts/generate_test_data.go <<EOF
package main
EOF

# empty docker compose file
: > docker-compose.yaml

echo "project structure ready"
