// Package telemetry implementa a camada unificada de medição e benchmark observacional
// de desempenho do computador (CPU, GPU, Memória e Sensores Térmicos).
//
// Princípios:
// 1. Não destrutivo e observacional: nunca executa stress tests ou overclock.
// 2. Honestidade de hardware: se um sensor não existir ou não estiver exposto,
//    reporta expressamente "indisponível", nunca inventando ou estimando valores.
// 3. 100% local e privado: nenhuma telemetria é enviada para fora da máquina.
package telemetry

import (
	"time"
)

// ProcessMetric representa o uso por um processo individual.
type ProcessMetric struct {
	PID     int     `json:"pid"`
	Name    string  `json:"name"`
	Percent float64 `json:"percent"`
	Memory  uint64  `json:"memoryBytes,omitempty"`
}

// MetricSample é uma amostra instantânea do estado da máquina no segundo T.
type MetricSample struct {
	Timestamp          time.Time       `json:"timestamp"`
	CPUUsagePercent    float64         `json:"cpuUsagePercent"`
	CPUFrequencyMHz    float64         `json:"cpuFrequencyMHz"`
	CPUTempCelsius     *float64        `json:"cpuTempCelsius,omitempty"`
	RAMUsedMB          float64         `json:"ramUsedMB"`
	RAMTotalMB         float64         `json:"ramTotalMB"`
	GPUUsagePercent    float64         `json:"gpuUsagePercent"`
	GPUMemoryUsedMB    float64         `json:"gpuMemoryUsedMB"`
	GPUMemoryTotalMB   float64         `json:"gpuMemoryTotalMB"`
	GPUTempCelsius     *float64        `json:"gpuTempCelsius,omitempty"`
	ThermalThrottling  bool            `json:"thermalThrottling"`
	TopProcessesCPU    []ProcessMetric `json:"topProcessesCpu,omitempty"`
	TopProcessGPU      string          `json:"topProcessGpu,omitempty"`
}

// HardwareStaticInfo contém informações estáticas de identificação do hardware.
type HardwareStaticInfo struct {
	CPUName        string `json:"cpuName"`
	PhysicalCores  int    `json:"physicalCores"`
	LogicalCores   int    `json:"logicalCores"`
	GPUName        string `json:"gpuName"`
	TotalRAMMB     float64 `json:"totalRamMb"`
	TotalGPUMemMB  float64 `json:"totalGpuMemMb"`
}

// BenchmarkReport consolida uma sessão de amostragem de benchmark (ex: 60 segundos).
type BenchmarkReport struct {
	ID                     string          `json:"id"`
	Stage                  string          `json:"stage"` // "before" | "after"
	DurationSeconds        int             `json:"durationSeconds"`
	SampleCount            int             `json:"sampleCount"`
	StartedAt              time.Time       `json:"startedAt"`
	FinishedAt             time.Time       `json:"finishedAt"`
	
	// CPU
	CPUName                string          `json:"cpuName"`
	PhysicalCores          int             `json:"physicalCores"`
	LogicalCores           int             `json:"logicalCores"`
	CPUUsageAvg            float64         `json:"cpuUsageAvg"`
	CPUUsagePeak           float64         `json:"cpuUsagePeak"`
	CPUUsageMin            float64         `json:"cpuUsageMin"`
	CPUFrequencyAvgMHz     float64         `json:"cpuFrequencyAvgMhz"`
	CPUTempAvailable       bool            `json:"cpuTempAvailable"`
	CPUTempAvg             *float64        `json:"cpuTempAvg,omitempty"`
	CPUTempPeak            *float64        `json:"cpuTempPeak,omitempty"`
	
	// Memória RAM
	RAMUsedAvgMB           float64         `json:"ramUsedAvgMb"`
	RAMUsedPeakMB          float64         `json:"ramUsedPeakMb"`
	RAMTotalMB             float64         `json:"ramTotalMb"`

	// GPU
	GPUName                string          `json:"gpuName"`
	GPUUsageAvg            float64         `json:"gpuUsageAvg"`
	GPUUsagePeak           float64         `json:"gpuUsagePeak"`
	GPUMemoryUsedAvgMB     float64         `json:"gpuMemoryUsedAvgMb"`
	GPUMemoryTotalMB       float64         `json:"gpuMemoryTotalMb"`
	GPUTempAvailable       bool            `json:"gpuTempAvailable"`
	GPUTempAvg             *float64        `json:"gpuTempAvg,omitempty"`
	GPUTempPeak            *float64        `json:"gpuTempPeak,omitempty"`

	// Sensores e Limites
	ThermalThrottled       bool            `json:"thermalThrottled"`
	TopProcessesCPU        []ProcessMetric `json:"topProcessesCpu,omitempty"`
	TopProcessGPU          string          `json:"topProcessGpu,omitempty"`
	HardwareHonestyNotice  string          `json:"hardwareHonestyNotice"`
}

// BenchmarkComparison compara as medições antes e depois da aplicação de um perfil.
type BenchmarkComparison struct {
	ProfileKey           string          `json:"profileKey"`
	BatchID              string          `json:"batchId"`
	Before               BenchmarkReport `json:"before"`
	After                BenchmarkReport `json:"after"`
	DeltaCPUUsageAvg     float64         `json:"deltaCpuUsageAvg"`
	DeltaCPUTempAvg      *float64        `json:"deltaCpuTempAvg,omitempty"`
	DeltaGPUUsageAvg     float64         `json:"deltaGpuUsageAvg"`
	DeltaGPUMemoryAvgMB  float64         `json:"deltaGpuMemoryAvgMb"`
	DeltaGPUTempAvg      *float64        `json:"deltaGpuTempAvg,omitempty"`
	ThrottlingResolved   bool            `json:"throttlingResolved"`
	ThrottlingOccurred   bool            `json:"throttlingOccurred"`
	Disclaimer           string          `json:"disclaimer"`
}
