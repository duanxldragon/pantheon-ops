@echo off
REM Pantheon Base Backend Development Startup Script
REM This script starts the backend server in development mode with GORM AutoMigrate enabled

echo ========================================
echo Starting Pantheon Base Backend (Dev Mode)
echo ========================================
echo.

REM Set environment variables for development
set PANTHEON_AUTO_MIGRATE=true
set PANTHEON_ENV=development

echo Environment:
echo   - PANTHEON_AUTO_MIGRATE=%PANTHEON_AUTO_MIGRATE%
echo   - PANTHEON_ENV=%PANTHEON_ENV%
echo.

REM Check if database is configured
if "%PANTHEON_DSN%"=="" (
    echo Warning: PANTHEON_DSN is not set!
    echo Please set it before running this script, e.g.:
    echo   set PANTHEON_DSN=root:dev_password_change_me@tcp(localhost:3306)/pantheon_base?charset=utf8mb4^&parseTime=True^&loc=Local
    echo.
    pause
    exit /b 1
)

echo Starting backend server...
echo.

REM Start the backend server
go run ./cmd/server/main.go

if errorlevel 1 (
    echo.
    echo Error: Failed to start backend server
    pause
    exit /b 1
)
