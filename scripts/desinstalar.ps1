# ==============================================================================
# OPTIMIZER — Desinstalador Limpo
# Remove o Optimizer e seus atalhos com segurança.
# ==============================================================================

param(
    [switch]$ApagarHistorico,
    [switch]$Silent
)

$ErrorActionPreference = "Stop"

if (-not $Silent) {
    Write-Host ""
    Write-Host "  ========================================================" -ForegroundColor DarkCyan
    Write-Host "     ⚡ OPTIMIZER — Desinstalador do Sistema" -ForegroundColor Cyan
    Write-Host "  ========================================================" -ForegroundColor DarkCyan
    Write-Host ""
}

# 1. Encerra processos do Optimizer se estiverem em execução
$procs = Get-Process Optimizer -ErrorAction SilentlyContinue
if ($procs) {
    if (-not $Silent) { Write-Host "  [1/3] Encerrando processos ativos do Optimizer..." -ForegroundColor Yellow }
    $procs | Stop-Process -Force
    Start-Sleep -Milliseconds 300
}

# 2. Remove pasta de instalação
$dest = "$env:LOCALAPPDATA\Programs\Optimizer"
if (Test-Path $dest) {
    if (-not $Silent) { Write-Host "  [2/3] Removendo arquivos de instalação em $dest..." -ForegroundColor Yellow }
    Remove-Item $dest -Recurse -Force
}

# 3. Remove atalhos do Menu Iniciar e Desktop
if (-not $Silent) { Write-Host "  [3/3] Removendo atalhos do Menu Iniciar e Desktop..." -ForegroundColor Yellow }
foreach ($lnk in @("$env:APPDATA\Microsoft\Windows\Start Menu\Programs\Optimizer.lnk",
                   ([Environment]::GetFolderPath('Desktop') + "\Optimizer.lnk"))) {
    if (Test-Path $lnk) {
        Remove-Item $lnk -Force
    }
}

# 4. Histórico
if ($ApagarHistorico) {
    $h = "$env:LOCALAPPDATA\Optimizer"
    if (Test-Path $h) {
        Remove-Item $h -Recurse -Force
        if (-not $Silent) { Write-Host "  ✔ Histórico apagado: $h" -ForegroundColor DarkGray }
    }
} else {
    if (-not $Silent) {
        Write-Host "  ℹ Histórico de reversão preservado em $env:LOCALAPPDATA\Optimizer" -ForegroundColor DarkGray
        Write-Host "    (Para apagar o histórico, execute com -ApagarHistorico)" -ForegroundColor DarkGray
    }
}

if (-not $Silent) {
    Write-Host ""
    Write-Host "  ✔ Optimizer desinstalado com sucesso!" -ForegroundColor Green
    Write-Host ""
}
