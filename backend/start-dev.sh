#!/bin/bash

# Pantheon Base Backend Development Startup Script
# This script starts the backend server in development mode with GORM AutoMigrate enabled

echo "========================================"
echo "Starting Pantheon Base Backend (Dev Mode)"
echo "========================================"
echo ""

# Set environment variables for development
export PANTHEON_AUTO_MIGRATE=true
export PANTHEON_ENV=development

echo "Environment:"
echo "  - PANTHEON_AUTO_MIGRATE=$PANTHEON_AUTO_MIGRATE"
echo "  - PANTHEON_ENV=$PANTHEON_ENV"
echo ""

# Check if database is configured
if [[ -z "$PANTHEON_DSN" ]]; then
    echo "Warning: PANTHEON_DSN is not set!"
    echo "Please set it before running this script, e.g.:"
    echo "  export PANTHEON_DSN='root:dev_password_change_me@tcp(localhost:3306)/pantheon_base?charset=utf8mb4&parseTime=True&loc=Local'"
    echo ""
    read -p "Press Enter to continue..."
    exit 1
fi

echo "Starting backend server..."
echo ""

# Start the backend server
go run ./cmd/server/main.go

if [[ $? -ne 0 ]]; then
    echo ""
    echo "Error: Failed to start backend server" >&2
    read -p "Press Enter to continue..."
    exit 1
fi
