@echo off
echo ============================================
echo  Jevon CRM — Production Build
echo ============================================

:: Переходим в папку фронтенда
cd /d "%~dp0..\jevon-frontend"
if not exist "package.json" (
    echo ERROR: package.json not found. Check frontend folder path.
    pause & exit /b 1
)

:: Собираем React
echo.
echo [1/3] Building React frontend...
call npm run build
if errorlevel 1 (
    echo ERROR: npm build failed
    pause & exit /b 1
)
echo     OK - built to ../jevon-backend/web/

:: Переходим в папку бэкенда
cd /d "%~dp0"
if not exist "go.mod" (
    echo ERROR: go.mod not found. Check backend folder path.
    pause & exit /b 1
)

:: Проверяем что web/ создалась
if not exist "web\index.html" (
    echo ERROR: web/index.html not found after npm build
    pause & exit /b 1
)

:: Собираем Go бинарник
echo.
echo [2/3] Building Go binary...
go build -tags prod -ldflags="-s -w" -o jevon.exe ./cmd/api/
if errorlevel 1 (
    echo ERROR: go build failed
    pause & exit /b 1
)
echo     OK - jevon.exe created

echo.
echo [3/3] Done!
echo ============================================
echo  Run: jevon.exe
echo  Open: http://localhost:8181
echo ============================================
pause