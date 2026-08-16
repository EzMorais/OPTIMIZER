# Remove o Optimizer desta máquina.
#
# Importante: isto apaga o app, NÃO desfaz as otimizações aplicadas. Desfaça
# antes, pelo próprio app (botão "Desfazer tudo") ou por
# `optimizerctl desfazer --tudo`. O histórico em %LOCALAPPDATA%\Optimizer é
# preservado, a não ser que você use -ApagarHistorico.

param([switch]$ApagarHistorico)

$ErrorActionPreference = "Stop"

Get-Process Optimizer -ErrorAction SilentlyContinue | Stop-Process -Force

$dest = "$env:LOCALAPPDATA\Programs\Optimizer"
if (Test-Path $dest) { Remove-Item $dest -Recurse -Force; Write-Host "Removido: $dest" }

foreach ($lnk in @("$env:APPDATA\Microsoft\Windows\Start Menu\Programs\Optimizer.lnk",
                   ([Environment]::GetFolderPath('Desktop') + "\Optimizer.lnk"))) {
    if (Test-Path $lnk) { Remove-Item $lnk -Force; Write-Host "Atalho removido: $lnk" }
}

if ($ApagarHistorico) {
    $h = "$env:LOCALAPPDATA\Optimizer"
    if (Test-Path $h) { Remove-Item $h -Recurse -Force; Write-Host "Histórico apagado: $h" }
} else {
    Write-Host "Histórico preservado em $env:LOCALAPPDATA\Optimizer" -ForegroundColor Yellow
}
