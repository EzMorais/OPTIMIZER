# ==============================================================================
# OPTIMIZER — Instalador Visual & Automatizado
# Compila e instala o Optimizer com interface gráfica moderna, sem exigir admin.
# ==============================================================================

param(
    [switch]$Silent,
    [switch]$NoLaunch
)

$ErrorActionPreference = "Stop"
$OutputEncoding = [System.Text.Encoding]::UTF8
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

$raiz = Split-Path $PSScriptRoot -Parent
$dest = "$env:LOCALAPPDATA\Programs\Optimizer"

# Se executado com -Silent ou em console não interativo, executa via terminal estilizado
if ($Silent -or ($Host.Name -notmatch "ConsoleHost|Windows PowerShell|Visual Studio Code")) {
    Write-Host ""
    Write-Host "  ========================================================" -ForegroundColor DarkCyan
    Write-Host "     OPTIMIZER -- Instalador de Alta Performance" -ForegroundColor Cyan
    Write-Host "  ========================================================" -ForegroundColor DarkCyan
    Write-Host ""

    New-Item -ItemType Directory -Path $dest -Force | Out-Null

    Write-Host "  [1/4] Compilando Optimizer.exe (Interface Grafica)..." -ForegroundColor Yellow
    Push-Location $raiz
    try {
        go build -tags "desktop,production" -ldflags "-H windowsgui -s -w" -o "$dest\Optimizer.exe" ./cmd/optimizerui
        if ($LASTEXITCODE -ne 0) { throw "Falha na compilacao do Optimizer.exe" }
        
        Write-Host "  [2/4] Compilando optimizerctl.exe (CLI)..." -ForegroundColor Yellow
        go build -ldflags "-s -w" -o "$dest\optimizerctl.exe" ./cmd/optimizerctl
        if ($LASTEXITCODE -ne 0) { throw "Falha na compilacao do optimizerctl.exe" }
    }
    finally {
        Pop-Location
    }

    Write-Host "  [3/4] Configurando icones e arquivos..." -ForegroundColor Yellow
    if (Test-Path "$raiz\cmd\optimizerui\icon.ico") {
        Copy-Item "$raiz\cmd\optimizerui\icon.ico" "$dest\icon.ico" -Force
    }

    Write-Host "  [4/4] Criando atalhos no Menu Iniciar e Desktop..." -ForegroundColor Yellow
    $ws = New-Object -ComObject WScript.Shell
    $iconPath = if (Test-Path "$dest\icon.ico") { "$dest\icon.ico" } else { "$dest\Optimizer.exe" }

    $desktopFolder = [Environment]::GetFolderPath('Desktop')
    $desktopLnk = Join-Path $desktopFolder "Optimizer.lnk"
    $startLnk = "$env:APPDATA\Microsoft\Windows\Start Menu\Programs\Optimizer.lnk"

    foreach ($lnk in @($startLnk, $desktopLnk)) {
        $s = $ws.CreateShortcut($lnk)
        $s.TargetPath = "$dest\Optimizer.exe"
        $s.WorkingDirectory = $dest
        $s.IconLocation = "$iconPath,0"
        $s.Description = "Optimizer - Otimizador de PC que mede antes de mexer e desfaz tudo"
        $s.Save()
    }

    Write-Host ""
    Write-Host "  [OK] Instalacao concluida com sucesso em: $dest" -ForegroundColor Green
    Write-Host ""
    exit 0
}

# Interface Gráfica Moderna (WPF / XAML)
Add-Type -AssemblyName PresentationFramework, PresentationCore, WindowsBase, System.Drawing

$xaml = @"
<Window xmlns="http://schemas.microsoft.com/winfx/2006/xaml/presentation"
        xmlns:x="http://schemas.microsoft.com/winfx/2006/xaml"
        Title="Instalador do Optimizer"
        Height="540" Width="680"
        WindowStartupLocation="CenterScreen"
        ResizeMode="NoResize"
        WindowStyle="None"
        AllowsTransparency="True"
        Background="Transparent"
        FontFamily="Segoe UI, -apple-system, Roboto, sans-serif">
    <Window.Resources>
        <Style TargetType="Button" x:Key="ModernBtn">
            <Setter Property="Background" Value="#10b981"/>
            <Setter Property="Foreground" Value="#ffffff"/>
            <Setter Property="FontWeight" Value="SemiBold"/>
            <Setter Property="FontSize" Value="14"/>
            <Setter Property="Cursor" Value="Hand"/>
            <Setter Property="BorderThickness" Value="0"/>
            <Setter Property="Padding" Value="20,10"/>
            <Setter Property="Template">
                <Setter.Value>
                    <ControlTemplate TargetType="Button">
                        <Border x:Name="border" Background="{TemplateBinding Background}" CornerRadius="8" Padding="{TemplateBinding Padding}">
                            <ContentPresenter HorizontalAlignment="Center" VerticalAlignment="Center"/>
                        </Border>
                        <ControlTemplate.Triggers>
                            <Trigger Property="IsMouseOver" Value="True">
                                <Setter TargetName="border" Property="Background" Value="#059669"/>
                            </Trigger>
                            <Trigger Property="IsEnabled" Value="False">
                                <Setter TargetName="border" Property="Background" Value="#1f2937"/>
                                <Setter Property="Foreground" Value="#6b7280"/>
                            </Trigger>
                        </ControlTemplate.Triggers>
                    </ControlTemplate>
                </Setter.Value>
            </Setter>
        </Style>
        <Style TargetType="Button" x:Key="SecondaryBtn">
            <Setter Property="Background" Value="#1f2937"/>
            <Setter Property="Foreground" Value="#9ca3af"/>
            <Setter Property="FontWeight" Value="SemiBold"/>
            <Setter Property="FontSize" Value="13"/>
            <Setter Property="Cursor" Value="Hand"/>
            <Setter Property="BorderThickness" Value="1"/>
            <Setter Property="BorderBrush" Value="#374151"/>
            <Setter Property="Padding" Value="16,8"/>
            <Setter Property="Template">
                <Setter.Value>
                    <ControlTemplate TargetType="Button">
                        <Border x:Name="border" Background="{TemplateBinding Background}" BorderBrush="{TemplateBinding BorderBrush}" BorderThickness="1" CornerRadius="8" Padding="{TemplateBinding Padding}">
                            <ContentPresenter HorizontalAlignment="Center" VerticalAlignment="Center"/>
                        </Border>
                        <ControlTemplate.Triggers>
                            <Trigger Property="IsMouseOver" Value="True">
                                <Setter TargetName="border" Property="Background" Value="#374151"/>
                                <Setter Property="Foreground" Value="#f3f4f6"/>
                            </Trigger>
                        </ControlTemplate.Triggers>
                    </ControlTemplate>
                </Setter.Value>
            </Setter>
        </Style>
    </Window.Resources>

    <Border Background="#0b121b" BorderBrush="#10b981" BorderThickness="1.5" CornerRadius="16">
        <Grid>
            <Grid.RowDefinitions>
                <RowDefinition Height="46"/>
                <RowDefinition Height="*"/>
                <RowDefinition Height="74"/>
            </Grid.RowDefinitions>

            <!-- Barra de Titulo Customizada -->
            <Grid Grid.Row="0" Background="#0f172a" Margin="1.5,1.5,1.5,0">
                <Grid.ColumnDefinitions>
                    <ColumnDefinition Width="*"/>
                    <ColumnDefinition Width="46"/>
                </Grid.ColumnDefinitions>
                <StackPanel Orientation="Horizontal" VerticalAlignment="Center" Margin="18,0,0,0">
                    <Ellipse Width="10" Height="10" Fill="#10b981" Margin="0,0,8,0"/>
                    <TextBlock Text="Optimizer Setup" Foreground="#94a3b8" FontWeight="SemiBold" FontSize="13"/>
                </StackPanel>
                <Button Name="BtnCloseHeader" Grid.Column="1" Content="X" Foreground="#94a3b8" Background="Transparent" BorderThickness="0" FontSize="14" Cursor="Hand"/>
            </Grid>

            <!-- Conteudo Principal -->
            <Grid Grid.Row="1" Margin="32,20,32,10">
                <StackPanel VerticalAlignment="Center">
                    <!-- Header do Produto -->
                    <StackPanel Orientation="Horizontal" Margin="0,0,0,16">
                        <Border Width="60" Height="60" CornerRadius="14" Background="#111c2a" BorderBrush="#10b981" BorderThickness="1.5" Margin="0,0,18,0">
                            <TextBlock Text="OP" Foreground="#10b981" FontWeight="Bold" FontSize="22" HorizontalAlignment="Center" VerticalAlignment="Center"/>
                        </Border>
                        <StackPanel VerticalAlignment="Center">
                            <TextBlock Text="OPTIMIZER" Foreground="#ffffff" FontSize="24" FontWeight="Bold"/>
                            <TextBlock Text="Otimizador de PC de Alta Performance e Baixa Latência" Foreground="#10b981" FontSize="13" FontWeight="Medium" Margin="0,3,0,0"/>
                        </StackPanel>
                    </StackPanel>

                    <!-- Descricao / Recursos -->
                    <Border Background="#111a24" BorderBrush="#1e293b" BorderThickness="1" CornerRadius="10" Padding="16" Margin="0,0,0,18">
                        <Grid>
                            <Grid.ColumnDefinitions>
                                <ColumnDefinition Width="*"/>
                                <ColumnDefinition Width="*"/>
                            </Grid.ColumnDefinitions>
                            <StackPanel Grid.Column="0" Margin="0,0,10,0">
                                <TextBlock Text="[OK] 55 Otimizacoes Criteriosas" Foreground="#e2e8f0" FontSize="12" Margin="0,0,0,6"/>
                                <TextBlock Text="[OK] Historico com Desfazer Total" Foreground="#e2e8f0" FontSize="12" Margin="0,0,0,6"/>
                                <TextBlock Text="[OK] Teste de MTU sem Fragmentacao" Foreground="#e2e8f0" FontSize="12"/>
                            </StackPanel>
                            <StackPanel Grid.Column="1" Margin="10,0,0,0">
                                <TextBlock Text="[OK] Benchmark de DNS com 1-Click" Foreground="#e2e8f0" FontSize="12" Margin="0,0,0,6"/>
                                <TextBlock Text="[OK] Telemetria em Tempo Real (60 FPS)" Foreground="#e2e8f0" FontSize="12" Margin="0,0,0,6"/>
                                <TextBlock Text="[OK] 100% Livre de Bloatware de Fundo" Foreground="#e2e8f0" FontSize="12"/>
                            </StackPanel>
                        </Grid>
                    </Border>

                    <!-- Destino da Instalacao -->
                    <StackPanel Margin="0,0,0,16">
                        <TextBlock Text="Local de Instalacao:" Foreground="#64748b" FontSize="11" FontWeight="SemiBold" Margin="0,0,0,4"/>
                        <TextBlock Name="TxtDestino" Text="" Foreground="#94a3b8" FontSize="12" FontFamily="Consolas, monospace"/>
                    </StackPanel>

                    <!-- Status e Barra de Progresso -->
                    <StackPanel>
                        <TextBlock Name="TxtStatus" Text="Pronto para compilar e instalar." Foreground="#cbd5e1" FontSize="13" FontWeight="Medium" Margin="0,0,0,8"/>
                        <ProgressBar Name="Progresso" Height="8" Minimum="0" Maximum="100" Value="0" Background="#1e293b" Foreground="#10b981" BorderThickness="0"/>
                    </StackPanel>
                </StackPanel>
            </Grid>

            <!-- Rodape com Botoes -->
            <Border Grid.Row="2" Background="#0f172a" BorderBrush="#1e293b" BorderThickness="0,1,0,0" CornerRadius="0,0,14,14" Padding="24,14">
                <Grid>
                    <StackPanel Orientation="Horizontal" HorizontalAlignment="Left" VerticalAlignment="Center">
                        <TextBlock Name="TxtFooterInfo" Text="Instalacao local por usuario (sem privilegios excessivos)" Foreground="#64748b" FontSize="11"/>
                    </StackPanel>
                    <StackPanel Orientation="Horizontal" HorizontalAlignment="Right" VerticalAlignment="Center">
                        <Button Name="BtnCancelar" Content="Cancelar" Style="{StaticResource SecondaryBtn}" Margin="0,0,10,0"/>
                        <Button Name="BtnInstalar" Content="Instalar Agora" Style="{StaticResource ModernBtn}"/>
                        <Button Name="BtnIniciar" Content="Iniciar Optimizer" Style="{StaticResource ModernBtn}" Visibility="Collapsed"/>
                    </StackPanel>
                </Grid>
            </Border>
        </Grid>
    </Border>
</Window>
"@

$reader = [System.Xml.XmlReader]::Create([System.IO.StringReader]::new($xaml))
$window = [System.Windows.Markup.XamlReader]::Load($reader)

# Elementos
$btnCloseHeader = $window.FindName("BtnCloseHeader")
$btnCancelar = $window.FindName("BtnCancelar")
$btnInstalar = $window.FindName("BtnInstalar")
$btnIniciar = $window.FindName("BtnIniciar")
$txtStatus = $window.FindName("TxtStatus")
$txtDestino = $window.FindName("TxtDestino")
$progresso = $window.FindName("Progresso")
$txtFooterInfo = $window.FindName("TxtFooterInfo")

$txtDestino.Text = $dest

$btnCloseHeader.Add_Click({ $window.Close() })
$btnCancelar.Add_Click({ $window.Close() })

$btnInstalar.Add_Click({
    $btnInstalar.IsEnabled = $false
    $btnCancelar.IsEnabled = $false

    $action = {
        param($w, $txtStatus, $progresso, $btnInstalar, $btnCancelar, $btnIniciar, $txtFooterInfo, $raiz, $dest)

        $updateUI = {
            param($msg, $val)
            $w.Dispatcher.Invoke([Action]{
                $txtStatus.Text = $msg
                $progresso.Value = $val
            })
        }

        try {
            & $updateUI "Criando diretorios de destino..." 10
            New-Item -ItemType Directory -Path $dest -Force | Out-Null
            Start-Sleep -Milliseconds 200

            & $updateUI "Compilando Optimizer.exe (Interface Grafica)..." 35
            Push-Location $raiz
            try {
                $process = Start-Process -FilePath "go" -ArgumentList "build", "-tags", "desktop,production", "-ldflags", "-H windowsgui -s -w", "-o", "$dest\Optimizer.exe", "./cmd/optimizerui" -NoNewWindow -PassThru -Wait
                if ($process.ExitCode -ne 0) { throw "Erro ao compilar Optimizer.exe" }

                & $updateUI "Compilando optimizerctl.exe (CLI)..." 65
                $processCtl = Start-Process -FilePath "go" -ArgumentList "build", "-ldflags", "-s -w", "-o", "$dest\optimizerctl.exe", "./cmd/optimizerctl" -NoNewWindow -PassThru -Wait
                if ($processCtl.ExitCode -ne 0) { throw "Erro ao compilar optimizerctl.exe" }
            }
            finally {
                Pop-Location
            }

            & $updateUI "Copiando ativos e icones em alta resolucao..." 80
            if (Test-Path "$raiz\cmd\optimizerui\icon.ico") {
                Copy-Item "$raiz\cmd\optimizerui\icon.ico" "$dest\icon.ico" -Force
            }

            & $updateUI "Registrando atalhos no Menu Iniciar e Desktop..." 95
            $ws = New-Object -ComObject WScript.Shell
            $iconPath = if (Test-Path "$dest\icon.ico") { "$dest\icon.ico" } else { "$dest\Optimizer.exe" }

            $desktopFolder = [Environment]::GetFolderPath('Desktop')
            $desktopLnk = Join-Path $desktopFolder "Optimizer.lnk"
            $startLnk = "$env:APPDATA\Microsoft\Windows\Start Menu\Programs\Optimizer.lnk"

            foreach ($lnk in @($startLnk, $desktopLnk)) {
                $s = $ws.CreateShortcut($lnk)
                $s.TargetPath = "$dest\Optimizer.exe"
                $s.WorkingDirectory = $dest
                $s.IconLocation = "$iconPath,0"
                $s.Description = "Optimizer - Otimizador de PC que mede antes de mexer e desfaz tudo"
                $s.Save()
            }

            & $updateUI "Instalacao concluida com sucesso!" 100

            $w.Dispatcher.Invoke([Action]{
                $txtStatus.Foreground = [System.Windows.Media.Brushes]::LimeGreen
                $txtFooterInfo.Text = "Pronto para uso. Atalhos criados no Desktop e Menu Iniciar."
                $btnInstalar.Visibility = [System.Windows.Visibility]::Collapsed
                $btnCancelar.Content = "Fechar"
                $btnCancelar.IsEnabled = $true
                $btnIniciar.Visibility = [System.Windows.Visibility]::Visible
                $btnIniciar.IsEnabled = $true
            })
        }
        catch {
            $err = $_.Exception.Message
            $w.Dispatcher.Invoke([Action]{
                $txtStatus.Text = "Falha na instalacao: $err"
                $txtStatus.Foreground = [System.Windows.Media.Brushes]::Tomato
                $btnCancelar.IsEnabled = $true
                $btnInstalar.IsEnabled = $true
            })
        }
    }

    [System.Threading.ThreadPool]::QueueUserWorkItem([System.Threading.WaitCallback]{
        & $action $window $txtStatus $progresso $btnInstalar $btnCancelar $btnIniciar $txtFooterInfo $raiz $dest
    }) | Out-Null
})

$btnIniciar.Add_Click({
    if (Test-Path "$dest\Optimizer.exe") {
        Start-Process "$dest\Optimizer.exe"
    }
    $window.Close()
})

$window.ShowDialog() | Out-Null
