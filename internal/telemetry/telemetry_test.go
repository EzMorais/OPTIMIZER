package telemetry

import (
	"context"
	"testing"
	"time"
)

func TestRunBenchmarkAveragesAndPeaks(t *testing.T) {
	temp1 := 45.0
	temp2 := 55.0
	gpuTemp1 := 50.0
	gpuTemp2 := 60.0

	mock := &MockProvider{
		HardwareInfo: HardwareStaticInfo{
			CPUName:       "Test Core i7",
			PhysicalCores: 8,
			LogicalCores:  16,
			GPUName:       "Test RTX 4070",
			TotalRAMMB:    16384,
			TotalGPUMemMB: 8192,
		},
		Samples: []MetricSample{
			{
				CPUUsagePercent:  10.0,
				CPUFrequencyMHz:  3600,
				CPUTempCelsius:   &temp1,
				RAMUsedMB:        4000,
				RAMTotalMB:       16384,
				GPUUsagePercent:  20.0,
				GPUMemoryUsedMB:  2000,
				GPUMemoryTotalMB: 8192,
				GPUTempCelsius:   &gpuTemp1,
				TopProcessesCPU: []ProcessMetric{
					{Name: "code.exe", Percent: 5.0},
				},
			},
			{
				CPUUsagePercent:  30.0,
				CPUFrequencyMHz:  4000,
				CPUTempCelsius:   &temp2,
				RAMUsedMB:        6000,
				RAMTotalMB:       16384,
				GPUUsagePercent:  40.0,
				GPUMemoryUsedMB:  4000,
				GPUMemoryTotalMB: 8192,
				GPUTempCelsius:   &gpuTemp2,
				TopProcessesCPU: []ProcessMetric{
					{Name: "code.exe", Percent: 15.0},
				},
			},
		},
	}

	collector := NewCollector(mock)
	ctx := context.Background()

	var progressCount int
	report, err := collector.RunBenchmark(ctx, "before", 20*time.Millisecond, 10*time.Millisecond, func(curr, total int, last MetricSample) {
		progressCount++
	})
	if err != nil {
		t.Fatalf("erro inesperado no benchmark: %v", err)
	}

	if report.SampleCount != 2 {
		t.Errorf("esperado 2 amostras, obteve %d", report.SampleCount)
	}
	if report.CPUUsageAvg != 20.0 {
		t.Errorf("esperado média CPU 20.0%%, obteve %.1f%%", report.CPUUsageAvg)
	}
	if report.CPUUsagePeak != 30.0 {
		t.Errorf("esperado pico CPU 30.0%%, obteve %.1f%%", report.CPUUsagePeak)
	}
	if report.CPUUsageMin != 10.0 {
		t.Errorf("esperado mínimo CPU 10.0%%, obteve %.1f%%", report.CPUUsageMin)
	}
	if report.RAMUsedAvgMB != 5000 {
		t.Errorf("esperado média RAM 5000MB, obteve %.0fMB", report.RAMUsedAvgMB)
	}
	if report.GPUUsageAvg != 30.0 {
		t.Errorf("esperado média GPU 30.0%%, obteve %.1f%%", report.GPUUsageAvg)
	}
	if !report.CPUTempAvailable || report.CPUTempAvg == nil || *report.CPUTempAvg != 50.0 {
		t.Errorf("esperado média de temperatura CPU 50.0C, obteve %v", report.CPUTempAvg)
	}
	if !report.GPUTempAvailable || report.GPUTempAvg == nil || *report.GPUTempAvg != 55.0 {
		t.Errorf("esperado média de temperatura GPU 55.0C, obteve %v", report.GPUTempAvg)
	}
	if progressCount != 2 {
		t.Errorf("esperado 2 callbacks de progresso, obteve %d", progressCount)
	}
}

func TestRunBenchmarkSensorsUnavailable(t *testing.T) {
	mock := &MockProvider{
		Samples: []MetricSample{
			{
				CPUUsagePercent: 12.0,
				CPUTempCelsius:  nil, // Sensor indisponível
				GPUTempCelsius:  nil, // Sensor indisponível
			},
			{
				CPUUsagePercent: 14.0,
				CPUTempCelsius:  nil,
				GPUTempCelsius:  nil,
			},
		},
	}

	collector := NewCollector(mock)
	report, err := collector.RunBenchmark(context.Background(), "before", 20*time.Millisecond, 10*time.Millisecond, nil)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}

	if report.CPUTempAvailable {
		t.Error("CPUTempAvailable deveria ser false quando sensores não estão presentes")
	}
	if report.CPUTempAvg != nil {
		t.Errorf("CPUTempAvg deveria ser nil, obteve %v", report.CPUTempAvg)
	}
	if report.GPUTempAvailable {
		t.Error("GPUTempAvailable deveria ser false quando sensores não estão presentes")
	}
	if report.HardwareHonestyNotice == "" {
		t.Error("HardwareHonestyNotice deveria conter aviso sobre sensores indisponíveis")
	}
}

func TestRunBenchmarkCancellation(t *testing.T) {
	mock := &MockProvider{
		CollectDelay: 50 * time.Millisecond,
		Samples: []MetricSample{
			{CPUUsagePercent: 10.0},
			{CPUUsagePercent: 15.0},
		},
	}

	collector := NewCollector(mock)
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := collector.RunBenchmark(ctx, "before", 5*time.Second, 50*time.Millisecond, nil)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("esperado erro de cancelamento de contexto")
	}
	if elapsed > 300*time.Millisecond {
		t.Errorf("coleta demorou muito para cancelar: %v", elapsed)
	}
}

func TestCompareBenchmarks(t *testing.T) {
	tempBefore := 60.0
	tempAfter := 55.0

	before := BenchmarkReport{
		CPUUsageAvg:        25.0,
		GPUUsageAvg:        40.0,
		GPUMemoryUsedAvgMB: 3000,
		CPUTempAvg:         &tempBefore,
		ThermalThrottled:   true,
	}

	after := BenchmarkReport{
		CPUUsageAvg:        20.0,
		GPUUsageAvg:        35.0,
		GPUMemoryUsedAvgMB: 2800,
		CPUTempAvg:         &tempAfter,
		ThermalThrottled:   false,
	}

	comp := CompareBenchmarks("jogo", "batch-123", before, after)

	if comp.DeltaCPUUsageAvg != -5.0 {
		t.Errorf("esperado delta CPU -5.0, obteve %.1f", comp.DeltaCPUUsageAvg)
	}
	if comp.DeltaGPUUsageAvg != -5.0 {
		t.Errorf("esperado delta GPU -5.0, obteve %.1f", comp.DeltaGPUUsageAvg)
	}
	if comp.DeltaGPUMemoryAvgMB != -200 {
		t.Errorf("esperado delta Memória GPU -200MB, obteve %.0f", comp.DeltaGPUMemoryAvgMB)
	}
	if comp.DeltaCPUTempAvg == nil || *comp.DeltaCPUTempAvg != -5.0 {
		t.Errorf("esperado delta Temp CPU -5.0C, obteve %v", comp.DeltaCPUTempAvg)
	}
	if !comp.ThrottlingResolved {
		t.Error("esperado ThrottlingResolved = true")
	}
}
