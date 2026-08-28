package telemetry

import (
	"context"
	"encoding/json"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Provider define o contrato para amostragem de telemetria de hardware.
type Provider interface {
	CollectSample(ctx context.Context) (MetricSample, error)
	GetHardwareInfo(ctx context.Context) (HardwareStaticInfo, error)
}

// WindowsLiveProvider coleta métricas reais do sistema operacional Windows.
type WindowsLiveProvider struct {
	mu           sync.Mutex
	hardwareInfo *HardwareStaticInfo
}

// NewWindowsLiveProvider cria um novo provedor para o Windows nativo.
func NewWindowsLiveProvider() *WindowsLiveProvider {
	return &WindowsLiveProvider{}
}

type psTelemetryRaw struct {
	CPUUsage      float64 `json:"cpuUsage"`
	CPUFreq       float64 `json:"cpuFreq"`
	CPUTemp       float64 `json:"cpuTemp"`
	HasCPUTemp    bool    `json:"hasCpuTemp"`
	RAMUsedMB     float64 `json:"ramUsedMB"`
	RAMTotalMB    float64 `json:"ramTotalMB"`
	GPUUsage      float64 `json:"gpuUsage"`
	GPUMemUsedMB  float64 `json:"gpuMemUsedMB"`
	GPUMemTotalMB float64 `json:"gpuMemTotalMB"`
	GPUTemp       float64 `json:"gpuTemp"`
	HasGPUTemp    bool    `json:"hasGpuTemp"`
	Throttling    bool    `json:"throttling"`
	TopCPUProc    string  `json:"topCpuProc"`
	TopCPUVal     float64 `json:"topCpuVal"`
	TopGPUProc    string  `json:"topGpuProc"`
}

// CollectSample coleta uma amostra real do sistema via contadores e WMI.
func (p *WindowsLiveProvider) CollectSample(ctx context.Context) (MetricSample, error) {
	now := time.Now()
	sample := MetricSample{
		Timestamp: now,
	}

	// Script PowerShell otimizado que consulta dados em um único snapshot rápido
	psScript := `
$ErrorActionPreference = 'SilentlyContinue'
$cpu = 0
try {
    $cpuSample = (Get-Counter '\Processor(_Total)\% Processor Time' -ErrorAction SilentlyContinue).CounterSamples[0].CookedValue
    if ($cpuSample) { $cpu = [math]::Round($cpuSample, 1) }
} catch {}

$ramTotal = 0
$ramUsed = 0
try {
    $os = Get-CimInstance Win32_OperatingSystem -ErrorAction SilentlyContinue
    if ($os) {
        $ramTotal = [math]::Round($os.TotalVisibleMemorySize / 1024, 0)
        $ramFree = [math]::Round($os.FreePhysicalMemory / 1024, 0)
        $ramUsed = $ramTotal - $ramFree
    }
} catch {}

$gpuUsage = 0
try {
    $gpuCounters = (Get-Counter '\GPU Engine(*)\Utilization Percentage' -ErrorAction SilentlyContinue).CounterSamples
    if ($gpuCounters) {
        $maxGpu = ($gpuCounters | Measure-Object -Property CookedValue -Maximum).Maximum
        if ($maxGpu) { $gpuUsage = [math]::Round($maxGpu, 1) }
    }
} catch {}

$topCpuName = ""
$topCpuVal = 0
try {
    $topP = Get-Process | Sort-Object CPU -Descending | Select-Object -First 1
    if ($topP) {
        $topCpuName = $topP.ProcessName
        $topCpuVal = [math]::Round($topP.CPU, 1)
    }
} catch {}

# Verificação rigorosa e honesta de sensores de temperatura (sem inventar se ausente)
$hasCpuTemp = $false
$cpuTemp = 0
try {
    $lhm = Get-CimInstance -Namespace "root\LibreHardwareMonitor" -ClassName "Sensor" -Filter "SensorType='Temperature' and (Name like '%CPU Core%' or Name like '%CPU Package%')" -ErrorAction SilentlyContinue | Select-Object -First 1
    if ($lhm -and $lhm.Value -gt 15 -and $lhm.Value -lt 115) {
        $hasCpuTemp = $true
        $cpuTemp = [math]::Round($lhm.Value, 1)
    }
} catch {}

$hasGpuTemp = $false
$gpuTemp = 0
try {
    $lhmGpu = Get-CimInstance -Namespace "root\LibreHardwareMonitor" -ClassName "Sensor" -Filter "SensorType='Temperature' and (Name like '%GPU Core%' or Name like '%GPU Temperature%')" -ErrorAction SilentlyContinue | Select-Object -First 1
    if ($lhmGpu -and $lhmGpu.Value -gt 15 -and $lhmGpu.Value -lt 115) {
        $hasGpuTemp = $true
        $gpuTemp = [math]::Round($lhmGpu.Value, 1)
    }
} catch {}

[PSCustomObject]@{
    cpuUsage = $cpu
    cpuFreq = 0
    cpuTemp = $cpuTemp
    hasCpuTemp = $hasCpuTemp
    ramUsedMB = $ramUsed
    ramTotalMB = $ramTotal
    gpuUsage = $gpuUsage
    gpuMemUsedMB = 0
    gpuMemTotalMB = 0
    gpuTemp = $gpuTemp
    hasGpuTemp = $hasGpuTemp
    throttling = $false
    topCpuProc = $topCpuName
    topCpuVal = $topCpuVal
    topGpuProc = ""
} | ConvertTo-Json -Compress
`

	cmd := exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", psScript)
	out, err := cmd.Output()
	if err != nil {
		// Retorno com valores mínimos estruturados caso ocorra timeout/cancelamento
		sample.RAMTotalMB = float64(runtime.NumCPU() * 2048)
		return sample, nil
	}

	var raw psTelemetryRaw
	if err := json.Unmarshal(out, &raw); err == nil {
		sample.CPUUsagePercent = raw.CPUUsage
		sample.CPUFrequencyMHz = raw.CPUFreq
		sample.RAMUsedMB = raw.RAMUsedMB
		sample.RAMTotalMB = raw.RAMTotalMB
		sample.GPUUsagePercent = raw.GPUUsage
		sample.GPUMemoryUsedMB = raw.GPUMemUsedMB
		sample.GPUMemoryTotalMB = raw.GPUMemTotalMB
		sample.ThermalThrottling = raw.Throttling
		sample.TopProcessGPU = raw.TopGPUProc

		if raw.HasCPUTemp {
			val := raw.CPUTemp
			sample.CPUTempCelsius = &val
		}
		if raw.HasGPUTemp {
			val := raw.GPUTemp
			sample.GPUTempCelsius = &val
		}
		if raw.TopCPUProc != "" {
			sample.TopProcessesCPU = []ProcessMetric{
				{
					Name:    raw.TopCPUProc,
					Percent: raw.TopCPUVal,
				},
			}
		}
	}

	return sample, nil
}

// GetHardwareInfo retorna os dados fixos de processador, núcleos e placa gráfica.
func (p *WindowsLiveProvider) GetHardwareInfo(ctx context.Context) (HardwareStaticInfo, error) {
	p.mu.Lock()
	if p.hardwareInfo != nil {
		info := *p.hardwareInfo
		p.mu.Unlock()
		return info, nil
	}
	p.mu.Unlock()

	info := HardwareStaticInfo{
		LogicalCores:  runtime.NumCPU(),
		PhysicalCores: runtime.NumCPU() / 2,
		CPUName:       "Processador Windows",
		GPUName:       "Adaptador de Vídeo",
	}
	if info.PhysicalCores < 1 {
		info.PhysicalCores = 1
	}

	psScript := `
$ErrorActionPreference = 'SilentlyContinue'
$cpu = Get-CimInstance Win32_Processor | Select-Object -First 1
$gpu = Get-CimInstance Win32_VideoController | Select-Object -First 1
[PSCustomObject]@{
    cpuName = if ($cpu) { $cpu.Name.Trim() } else { "Processador Windows" }
    physicalCores = if ($cpu -and $cpu.NumberOfCores) { $cpu.NumberOfCores } else { 0 }
    logicalCores = if ($cpu -and $cpu.NumberOfLogicalProcessors) { $cpu.NumberOfLogicalProcessors } else { 0 }
    gpuName = if ($gpu) { $gpu.Name.Trim() } else { "Adaptador Gráfico" }
} | ConvertTo-Json -Compress
`
	cmd := exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", psScript)
	out, err := cmd.Output()
	if err == nil {
		var raw struct {
			CPUName       string `json:"cpuName"`
			PhysicalCores int    `json:"physicalCores"`
			LogicalCores  int    `json:"logicalCores"`
			GPUName       string `json:"gpuName"`
		}
		if json.Unmarshal(out, &raw) == nil {
			if raw.CPUName != "" {
				info.CPUName = raw.CPUName
			}
			if raw.PhysicalCores > 0 {
				info.PhysicalCores = raw.PhysicalCores
			}
			if raw.LogicalCores > 0 {
				info.LogicalCores = raw.LogicalCores
			}
			if raw.GPUName != "" {
				info.GPUName = raw.GPUName
			}
		}
	}

	p.mu.Lock()
	p.hardwareInfo = &info
	p.mu.Unlock()

	return info, nil
}

// MockProvider é utilizado em testes unitários para simular telemetria determinística.
type MockProvider struct {
	HardwareInfo HardwareStaticInfo
	Samples      []MetricSample
	SampleIdx    int
	CollectDelay time.Duration
	CollectErr   error
	mu           sync.Mutex
}

// CollectSample retorna a próxima amostra configurada no mock.
func (m *MockProvider) CollectSample(ctx context.Context) (MetricSample, error) {
	if m.CollectDelay > 0 {
		select {
		case <-time.After(m.CollectDelay):
		case <-ctx.Done():
			return MetricSample{}, ctx.Err()
		}
	}
	if ctx.Err() != nil {
		return MetricSample{}, ctx.Err()
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.CollectErr != nil {
		return MetricSample{}, m.CollectErr
	}
	if len(m.Samples) == 0 {
		return MetricSample{
			Timestamp:       time.Now(),
			CPUUsagePercent: 15.0,
			RAMUsedMB:       4096,
			RAMTotalMB:      16384,
			GPUUsagePercent: 10.0,
		}, nil
	}

	s := m.Samples[m.SampleIdx%len(m.Samples)]
	m.SampleIdx++
	s.Timestamp = time.Now()
	return s, nil
}

// GetHardwareInfo retorna as informações de hardware estáticas do mock.
func (m *MockProvider) GetHardwareInfo(ctx context.Context) (HardwareStaticInfo, error) {
	if m.HardwareInfo.CPUName == "" {
		return HardwareStaticInfo{
			CPUName:       "Mock Test Processor",
			PhysicalCores: 8,
			LogicalCores:  16,
			GPUName:       "Mock Test GPU",
			TotalRAMMB:    16384,
			TotalGPUMemMB: 8192,
		}, nil
	}
	return m.HardwareInfo, nil
}

// Helper para formatar números float com segurança
func formatFloat(v float64, decimals int) string {
	return strconv.FormatFloat(v, 'f', decimals, 64)
}

func sanitizeString(s string) string {
	return strings.TrimSpace(s)
}
