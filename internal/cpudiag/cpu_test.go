package cpudiag

import (
	"context"
	"testing"
)

type fakeProber struct {
	usage float64
}

func (f *fakeProber) GetCPUInfo(ctx context.Context) (CPUInfo, error) {
	return CPUInfo{
		TotalUsagePercent:    f.usage,
		PhysicalCores:        8,
		LogicalProcessors:    16,
		TemperatureAvailable: false,
		TopProcesses: []ProcessCPU{
			{PID: 1000, Name: "go.exe", CPUPercent: 24.5},
		},
		Interpretation: "Carga moderada.",
	}, nil
}

func TestCPUInfoFakeProber(t *testing.T) {
	fake := &fakeProber{usage: 42.5}
	info, err := fake.GetCPUInfo(context.Background())
	if err != nil {
		t.Fatalf("GetCPUInfo falhou: %v", err)
	}

	if info.TotalUsagePercent != 42.5 {
		t.Errorf("TotalUsagePercent = %.1f, esperado 42.5", info.TotalUsagePercent)
	}
	if info.PhysicalCores != 8 || info.LogicalProcessors != 16 {
		t.Errorf("Cores inesperados: %d/%d", info.PhysicalCores, info.LogicalProcessors)
	}
	if info.TemperatureAvailable {
		t.Error("TemperatureAvailable não deve ser true sem sensor confiável")
	}
}
