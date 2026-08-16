# Compila e instala o Optimizer nesta máquina, para teste.
#
# Instalação por usuário (%LOCALAPPDATA%), sem exigir administrador e sem
# serviço nenhum rodando em segundo plano — o app só roda quando é aberto.
# Para desinstalar: scripts\desinstalar.ps1

$ErrorActionPreference = "Stop"
$raiz = Split-Path $PSScriptRoot -Parent
$dest = "$env:LOCALAPPDATA\Programs\Optimizer"

Write-Host "Compilando o app..." -ForegroundColor Cyan
Push-Location $raiz
try {
    # -H windowsgui: sem janela de console atrás do app.
    go build -tags "desktop,production" -ldflags "-H windowsgui -s -w" -o "$dest\Optimizer.exe" ./cmd/optimizerui
    if ($LASTEXITCODE -ne 0) { throw "falha ao compilar o app" }

    Write-Host "Compilando a CLI interna..." -ForegroundColor Cyan
    go build -ldflags "-s -w" -o "$dest\optimizerctl.exe" ./cmd/optimizerctl
    if ($LASTEXITCODE -ne 0) { throw "falha ao compilar a CLI" }
}
finally { Pop-Location }

$ws = New-Object -ComObject WScript.Shell
foreach ($lnk in @("$env:APPDATA\Microsoft\Windows\Start Menu\Programs\Optimizer.lnk",
                   ([Environment]::GetFolderPath('Desktop') + "\Optimizer.lnk"))) {
    $s = $ws.CreateShortcut($lnk)
    $s.TargetPath = "$dest\Optimizer.exe"
    $s.WorkingDirectory = $dest
    $s.Description = "Optimizer - otimizador de PC que mede antes de mexer"
    $s.Save()
}

Write-Host "Instalado em $dest" -ForegroundColor Green
Write-Host "Atalhos criados no Menu Iniciar e na Área de Trabalho." -ForegroundColor Green
