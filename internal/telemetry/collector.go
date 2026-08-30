package telemetry

import (
	"context"
	"fmt"
	"math"
	"time"
)

// ProgressCallback recebe o progresso a cada segundo de amostragem.
type ProgressCallback func(currentSample int, totalSamples int, lastSample MetricSample)

// Collector orquestra a amostragem de telemetria e a geração de relatórios consolidados.
type Collector struct {
	Provider Provider
}

// NewCollector cria um novo coletor com o provedor especificado.
func NewCollector(p Provider) *Collector {
	if p == nil {
		p = NewWindowsLiveProvider()
	}
	return &Collector{Provider: p}
}

// RunBenchmark executa uma sessão observacional de medição com a duração especificada.
func (c *Collector) RunBenchmark(
	ctx context.Context,
	stage string,
	duration time.Duration,
	interval time.Duration,
	onProgress ProgressCallback,
) (BenchmarkReport, error) {
	if duration <= 0 {
		duration = 60 * time.Second
	}
	if interval <= 0 {
		interval = 1 * time.Second
	}

	totalExpectedSamples := int(duration / interval)
	if totalExpectedSamples < 1 {
		totalExpectedSamples = 1
	}

	startedAt := time.Now()
	report := BenchmarkReport{
		ID:              fmt.Sprintf("bench-%s-%d", stage, startedAt.UnixNano()),
		Stage:           stage,
		DurationSeconds: int(duration.Seconds()),
		StartedAt:       startedAt,
	}

	// Obter informações de hardware
	hwInfo, err := c.Provider.GetHardwareInfo(ctx)
	if err == nil {
		report.CPUName = hwInfo.CPUName
		report.PhysicalCores = hwInfo.PhysicalCores
		report.LogicalCores = hwInfo.LogicalCores
		report.GPUName = hwInfo.GPUName
		report.RAMTotalMB = hwInfo.TotalRAMMB
		report.GPUMemoryTotalMB = hwInfo.TotalGPUMemMB
	}

	var samples []MetricSample
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Primeira amostra imediata
	firstSample, err := c.Provider.CollectSample(ctx)
	if err == nil {
		samples = append(samples, firstSample)
		if onProgress != nil {
			onProgress(len(samples), totalExpectedSamples, firstSample)
		}
	}

	for len(samples) < totalExpectedSamples {
		select {
		case <-ctx.Done():
			// Cancelamento solicitado pelo usuário ou fechamento de tela
			report.FinishedAt = time.Now()
			if len(samples) > 0 {
				c.consolidateReport(&report, samples)
			}
			return report, ctx.Err()
		case <-ticker.C:
			s, err := c.Provider.CollectSample(ctx)
			if err != nil && ctx.Err() != nil {
				report.FinishedAt = time.Now()
				if len(samples) > 0 {
					c.consolidateReport(&report, samples)
				}
				return report, ctx.Err()
			}
			if err == nil {
				samples = append(samples, s)
				if onProgress != nil {
					onProgress(len(samples), totalExpectedSamples, s)
				}
			}
		}
	}

	report.FinishedAt = time.Now()
	c.consolidateReport(&report, samples)
	return report, nil
}

// consolidateReport calcula as médias, picos e consolida as métricas observadas.
func (c *Collector) consolidateReport(report *BenchmarkReport, samples []MetricSample) {
	report.SampleCount = len(samples)
	if len(samples) == 0 {
		return
	}

	var totalCPU, peakCPU float64
	minCPU := math.MaxFloat64
	var totalFreq float64
	var totalRAM, peakRAM, maxRAMTotal float64

	var totalGPU, peakGPU float64
	var totalGPUMem, peakGPUMem, maxGPUMemTotal float64

	var cpuTemps []float64
	var gpuTemps []float64

	type processAggregate struct {
		metric       ProcessMetric
		totalPercent float64
		samples      int
	}
	procMap := make(map[string]*processAggregate)

	for _, s := range samples {
		// CPU
		totalCPU += s.CPUUsagePercent
		if s.CPUUsagePercent > peakCPU {
			peakCPU = s.CPUUsagePercent
		}
		if s.CPUUsagePercent < minCPU {
			minCPU = s.CPUUsagePercent
		}
		totalFreq += s.CPUFrequencyMHz

		// RAM
		totalRAM += s.RAMUsedMB
		if s.RAMUsedMB > peakRAM {
			peakRAM = s.RAMUsedMB
		}
		if s.RAMTotalMB > maxRAMTotal {
			maxRAMTotal = s.RAMTotalMB
		}

		// GPU
		totalGPU += s.GPUUsagePercent
		if s.GPUUsagePercent > peakGPU {
			peakGPU = s.GPUUsagePercent
		}
		totalGPUMem += s.GPUMemoryUsedMB
		if s.GPUMemoryUsedMB > peakGPUMem {
			peakGPUMem = s.GPUMemoryUsedMB
		}
		if s.GPUMemoryTotalMB > maxGPUMemTotal {
			maxGPUMemTotal = s.GPUMemoryTotalMB
		}

		// Temperaturas
		if s.CPUTempCelsius != nil {
			cpuTemps = append(cpuTemps, *s.CPUTempCelsius)
		}
		if s.GPUTempCelsius != nil {
			gpuTemps = append(gpuTemps, *s.GPUTempCelsius)
		}

		// Throttling
		if s.ThermalThrottling {
			report.ThermalThrottled = true
		}

		// Processos CPU
		for _, p := range s.TopProcessesCPU {
			if p.Name != "" {
				key := fmt.Sprintf("%d:%s", p.PID, p.Name)
				agg := procMap[key]
				if agg == nil {
					agg = &processAggregate{metric: p}
					procMap[key] = agg
				}
				agg.totalPercent += clampPercent(p.Percent)
				agg.samples++
			}
		}
		if s.TopProcessGPU != "" {
			report.TopProcessGPU = s.TopProcessGPU
		}
	}

	n := float64(len(samples))
	report.CPUUsageAvg = roundFloat(totalCPU/n, 1)
	report.CPUUsagePeak = roundFloat(peakCPU, 1)
	if minCPU != math.MaxFloat64 {
		report.CPUUsageMin = roundFloat(minCPU, 1)
	}
	report.CPUFrequencyAvgMHz = roundFloat(totalFreq/n, 0)

	report.RAMUsedAvgMB = roundFloat(totalRAM/n, 0)
	report.RAMUsedPeakMB = roundFloat(peakRAM, 0)
	if report.RAMTotalMB == 0 {
		report.RAMTotalMB = roundFloat(maxRAMTotal, 0)
	}

	report.GPUUsageAvg = roundFloat(totalGPU/n, 1)
	report.GPUUsagePeak = roundFloat(peakGPU, 1)
	report.GPUMemoryUsedAvgMB = roundFloat(totalGPUMem/n, 0)
	if report.GPUMemoryTotalMB == 0 {
		report.GPUMemoryTotalMB = roundFloat(maxGPUMemTotal, 0)
	}

	// Médias de Temperatura CPU
	if len(cpuTemps) > 0 {
		report.CPUTempAvailable = true
		var sum, peak float64
		for _, t := range cpuTemps {
			sum += t
			if t > peak {
				peak = t
			}
		}
		avg := roundFloat(sum/float64(len(cpuTemps)), 1)
		report.CPUTempAvg = &avg
		report.CPUTempPeak = &peak
	} else {
		report.CPUTempAvailable = false
	}

	// Médias de Temperatura GPU
	if len(gpuTemps) > 0 {
		report.GPUTempAvailable = true
		var sum, peak float64
		for _, t := range gpuTemps {
			sum += t
			if t > peak {
				peak = t
			}
		}
		avg := roundFloat(sum/float64(len(gpuTemps)), 1)
		report.GPUTempAvg = &avg
		report.GPUTempPeak = &peak
	} else {
		report.GPUTempAvailable = false
	}

	if !report.CPUTempAvailable || !report.GPUTempAvailable {
		report.HardwareHonestyNotice = "Sensores de temperatura de hardware não expostos pelo driver nesta máquina. Nenhuma estimativa artificial foi gerada."
	}

	// Consolidação de processos
	for _, agg := range procMap {
		if agg.samples == 0 {
			continue
		}
		report.TopProcessesCPU = append(report.TopProcessesCPU, ProcessMetric{
			PID:     agg.metric.PID,
			Name:    agg.metric.Name,
			Memory:  agg.metric.Memory,
			Percent: clampPercent(roundFloat(agg.totalPercent/float64(agg.samples), 1)),
		})
	}
}

// CompareBenchmarks compara dois relatórios de benchmark (antes e depois).
func CompareBenchmarks(profileKey string, batchID string, before BenchmarkReport, after BenchmarkReport) BenchmarkComparison {
	comp := BenchmarkComparison{
		ProfileKey: profileKey,
		BatchID:    batchID,
		Before:     before,
		After:      after,
		Disclaimer: "Esta é uma medição observacional local realizada na sua máquina. Variações de temperatura e FPS dependem da carga do sistema e do aplicativo ou jogo em execução no momento.",
	}

	comp.DeltaCPUUsageAvg = roundFloat(after.CPUUsageAvg-before.CPUUsageAvg, 1)
	comp.DeltaGPUUsageAvg = roundFloat(after.GPUUsageAvg-before.GPUUsageAvg, 1)
	comp.DeltaGPUMemoryAvgMB = roundFloat(after.GPUMemoryUsedAvgMB-before.GPUMemoryUsedAvgMB, 0)

	if before.CPUTempAvg != nil && after.CPUTempAvg != nil {
		delta := roundFloat(*after.CPUTempAvg-*before.CPUTempAvg, 1)
		comp.DeltaCPUTempAvg = &delta
	}

	if before.GPUTempAvg != nil && after.GPUTempAvg != nil {
		delta := roundFloat(*after.GPUTempAvg-*before.GPUTempAvg, 1)
		comp.DeltaGPUTempAvg = &delta
	}

	if before.ThermalThrottled && !after.ThermalThrottled {
		comp.ThrottlingResolved = true
	} else if !before.ThermalThrottled && after.ThermalThrottled {
		comp.ThrottlingOccurred = true
	}

	return comp
}

func roundFloat(val float64, precision int) float64 {
	ratio := math.Pow(10, float64(precision))
	return math.Round(val*ratio) / ratio
}
