@echo off
setlocal
echo ========================================================
echo   Compilando OPTIMIZER com tags de producao Wails...
echo ========================================================
echo.

echo [1/2] Compilando Optimizer.exe...
go build -tags "desktop,production" -ldflags "-H windowsgui -s -w" -o "%~dp0Optimizer.exe" ./cmd/optimizerui
if %ERRORLEVEL% neq 0 (
    echo [ERRO] Falha ao compilar Optimizer.exe!
    pause
    exit /b %ERRORLEVEL%
)

echo [2/2] Compilando optimizerctl.exe...
go build -ldflags "-s -w" -o "%~dp0optimizerctl.exe" ./cmd/optimizerctl
if %ERRORLEVEL% neq 0 (
    echo [ERRO] Falha ao compilar optimizerctl.exe!
    pause
    exit /b %ERRORLEVEL%
)

echo.
echo [OK] Compilacao concluida com sucesso na raiz do projeto:
echo   - %~dp0Optimizer.exe
echo   - %~dp0optimizerctl.exe
echo.
pause

