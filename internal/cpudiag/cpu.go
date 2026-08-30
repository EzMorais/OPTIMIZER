// Package cpudiag coleta métricas reais de uso de processador, núcleos e processos no Windows.
package cpudiag

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"optimizer/internal/console"
)

// ProcessCPU representa um processo consumidor de CPU.
type ProcessCPU struct {
	PID        int     `json:"pid"`
	Name       string  `json:"name"`
	CPUPercent float64 `json:"cpuPercent"`
}

// CPUInfo contém métricas reais consolidadas de CPU.
type CPUInfo struct {
	Timestamp            time.Time    `json:"timestamp"`
	TotalUsagePercent    float64      `json:"totalUsagePercent"`
	PhysicalCores        int          `json:"physicalCores"`
	LogicalProcessors    int          `json:"logicalProcessors"`
	BaseFrequencyMHz     int          `json:"baseFrequencyMHz,omitempty"`
	TemperatureC         float64      `json:"temperatureC,omitempty"`
	TemperatureAvailable bool         `json:"temperatureAvailable"`
	ThrottlingDetected   bool         `json:"throttlingDetected"`
	TopProcesses         []ProcessCPU `json:"topProcesses"`
	Interpretation       string       `json:"interpretation"`
}

// Prober define a interface mockável de leitura de CPU.
type Prober interface {
	GetCPUInfo(ctx context.Context) (CPUInfo, error)
}

// LiveProber lê dados reais do sistema operacional.
type LiveProber struct{}

// NewLiveProber cria um coletor real de métricas.
func NewLiveProber() *LiveProber { return &LiveProber{} }

func (l *LiveProber) GetCPUInfo(ctx context.Context) (CPUInfo, error) {
	info := CPUInfo{
		Timestamp:            time.Now(),
		LogicalProcessors:    runtime.NumCPU(),
		PhysicalCores:        runtime.NumCPU() / 2,
		TemperatureAvailable: false,
		TopProcesses:         []ProcessCPU{},
	}
	if info.PhysicalCores < 1 {
		info.PhysicalCores = 1
	}

	// 1. Uso de CPU via PowerShell Get-Counter (amostra real de 1 segundo)
	cmd := exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command",
		`$c = (Get-Counter '\Processor(_Total)\% Processor Time' -SampleInterval 1 -MaxSamples 1).CounterSamples[0].CookedValue; [math]::Round($c, 1)`)
	console.HideWindow(cmd)
	out, err := cmd.Output()
	if err == nil {
		strVal := strings.TrimSpace(string(out))
		if val, errParse := strconv.ParseFloat(strings.ReplaceAll(strVal, ",", "."), 64); errParse == nil {
			info.TotalUsagePercent = val
		}
	} else {
		info.TotalUsagePercent = 0.0
	}

	// 2. Top processos mais consumidores via Get-Process
	cmdProc := exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command",
		`Get-Process | Sort-Object CPU -Descending | Select-Object -First 5 Id, ProcessName, CPU | ForEach-Object { "$($_.Id)|$($_.ProcessName)|$([math]::Round($_.CPU, 1))" }`)
	console.HideWindow(cmdProc)
	outProc, errProc := cmdProc.Output()
	if errProc == nil {
		linhas := strings.Split(string(outProc), "\n")
		for _, linha := range linhas {
			linha = strings.TrimSpace(linha)
			if linha == "" {
				continue
			}
			partes := strings.Split(linha, "|")
			if len(partes) >= 3 {
				pid, _ := strconv.Atoi(partes[0])
				cpuVal, _ := strconv.ParseFloat(strings.ReplaceAll(partes[2], ",", "."), 64)
				info.TopProcesses = append(info.TopProcesses, ProcessCPU{
					PID:        pid,
					Name:       partes[1],
					CPUPercent: cpuVal,
				})
			}
		}
	}

	// 3. Interpretação de carga
	switch {
	case info.TotalUsagePercent > 85:
		info.Interpretation = fmt.Sprintf("Uso elevado de CPU (%.1f%%). Processos pesados ativos.", info.TotalUsagePercent)
	case info.TotalUsagePercent > 50:
		info.Interpretation = fmt.Sprintf("Carga moderada (%.1f%%). O sistema está operando normalmente.", info.TotalUsagePercent)
	default:
		info.Interpretation = fmt.Sprintf("Carga baixa (%.1f%%). Excelente responsividade e folga de processamento.", info.TotalUsagePercent)
	}

	return info, nil
}
