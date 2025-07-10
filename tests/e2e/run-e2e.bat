@echo off
echo [E2E] Starting E2E Test Runner...

REM Find project root directory
set PROJECT_ROOT=%~dp0..\..
cd /d "%PROJECT_ROOT%"

REM Verify we're in the right place
if not exist "go.mod" (
    echo [ERROR] Cannot find go.mod - not in project root
    echo [ERROR] Current directory: %CD%
    exit /b 1
)

echo [E2E] Project root found: %CD%

REM Configuration
set COMPOSE_FILE=tests\e2e\docker\docker-compose.e2e.yml
set PROJECT_NAME=weatherapi-e2e
set E2E_DIR=tests\e2e

REM Check if compose file exists
if not exist "%COMPOSE_FILE%" (
    echo [ERROR] Docker compose file not found: %COMPOSE_FILE%
    exit /b 1
)

echo [E2E] Using compose file: %COMPOSE_FILE%

REM Check command line argument
if "%1"=="" (
    echo E2E Test Runner for Weather API
    echo Usage: %0 ^<command^>
    echo.
    echo Commands:
    echo   setup       - Set up environment
    echo   test        - Run tests
    echo   full        - Setup + Test + Cleanup
    echo   cleanup     - Clean up
    echo   status      - Show status
    echo   logs        - Show logs
    exit /b 1
)

REM Execute commands directly
if "%1"=="setup" (
    echo [E2E] Setting up environment...
    
    REM Install Node dependencies
    echo [E2E] Installing Node.js dependencies...
    cd "%E2E_DIR%"
    if not exist "node_modules" npm install
    npx playwright install
    
    REM Return to project root
    cd "%PROJECT_ROOT%"
    
    REM Setup Docker
    echo [E2E] Setting up Docker services...
    docker-compose -f "%COMPOSE_FILE%" -p "%PROJECT_NAME%" down --volumes --remove-orphans
    docker-compose -f "%COMPOSE_FILE%" -p "%PROJECT_NAME%" build --no-cache
    docker-compose -f "%COMPOSE_FILE%" -p "%PROJECT_NAME%" up -d
    
    REM Wait for services
    echo [E2E] Waiting for services to be ready...
    timeout /t 60 /nobreak > nul
    echo [E2E] Checking if services are healthy...
    docker-compose -f "%COMPOSE_FILE%" -p "%PROJECT_NAME%" ps
    echo [E2E] Setup complete!
    goto end
)

if "%1"=="test" (
    echo [E2E] Running E2E tests...
    cd "%E2E_DIR%"
    npx playwright test
    if %errorlevel% neq 0 (
        echo [ERROR] Tests failed
        exit /b %errorlevel%
    )
    echo [E2E] Tests completed successfully!
    goto end
)

if "%1"=="full" (
    echo [E2E] Running full E2E cycle...
    
    REM SETUP PHASE
    echo [E2E] Setting up environment...
    
    REM Install Node dependencies
    echo [E2E] Installing Node.js dependencies...
    cd "%E2E_DIR%"
    if not exist "node_modules" npm install
    npx playwright install
    
    REM Return to project root
    cd "%PROJECT_ROOT%"
    
    REM Setup Docker
    echo [E2E] Setting up Docker services...
    docker-compose -f "%COMPOSE_FILE%" -p "%PROJECT_NAME%" down --volumes --remove-orphans
    docker-compose -f "%COMPOSE_FILE%" -p "%PROJECT_NAME%" build --no-cache
    docker-compose -f "%COMPOSE_FILE%" -p "%PROJECT_NAME%" up -d
    
    REM Wait for services
    echo [E2E] Waiting for services to be ready...
    timeout /t 60 /nobreak > nul
    echo [E2E] Checking if services are healthy...
    docker-compose -f "%COMPOSE_FILE%" -p "%PROJECT_NAME%" ps
    echo [E2E] Setup complete!
    
    REM TEST PHASE
    echo [E2E] Running E2E tests...
    cd "%E2E_DIR%"
    npx playwright test
    if %errorlevel% neq 0 (
        echo [ERROR] Tests failed
        cd "%PROJECT_ROOT%"
        docker-compose -f "%COMPOSE_FILE%" -p "%PROJECT_NAME%" down --volumes --remove-orphans
        exit /b %errorlevel%
    )
    echo [E2E] Tests completed successfully!
    
    REM CLEANUP PHASE
    cd "%PROJECT_ROOT%"
    echo [E2E] Cleaning up...
    docker-compose -f "%COMPOSE_FILE%" -p "%PROJECT_NAME%" down --volumes --remove-orphans
    echo [E2E] Cleanup complete!
    goto end
)

if "%1"=="cleanup" (
    echo [E2E] Cleaning up...
    docker-compose -f "%COMPOSE_FILE%" -p "%PROJECT_NAME%" down --volumes --remove-orphans
    echo [E2E] Cleanup complete!
    goto end
)

if "%1"=="status" (
    echo [E2E] Service status:
    docker-compose -f "%COMPOSE_FILE%" -p "%PROJECT_NAME%" ps
    goto end
)

if "%1"=="logs" (
    echo [E2E] Service logs:
    docker-compose -f "%COMPOSE_FILE%" -p "%PROJECT_NAME%" logs
    goto end
)

echo [ERROR] Unknown command: %1
exit /b 1

:end
echo [E2E] Done.
