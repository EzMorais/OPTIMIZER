package systemdiag_test

import (
	"testing"

	"optimizer/internal/systemdiag"
)

func TestObterTimerResolution(t *testing.T) {
	info, err := systemdiag.ObterTimerResolution()
	if err != nil {
		t.Logf("Aviso ao obter timer resolution: %v", err)
	}
	if info.MinResolutionMs <= 0 || info.MaxResolutionMs <= 0 || info.CurrentResolutionMs <= 0 {
		t.Errorf("Valores inválidos de timer resolution: %+v", info)
	}
}

func TestMedirSleepPrecision(t *testing.T) {
	res := systemdiag.MedirSleepPrecision(10)
	if len(res.Samples) != 10 {
		t.Errorf("Esperava 10 amostras, obteve %d", len(res.Samples))
	}
	if res.AverageMs <= 0 || res.JitterScore == "" {
		t.Errorf("Resultado inválido de medição: %+v", res)
	}
}
