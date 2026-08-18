package netdiag

import (
	"context"
	"net/netip"
	"testing"
	"time"
)

// fakeRTTProber simula respostas ICMP sem tocar a rede real.
type fakeRTTProber struct {
	rtts []int // um valor por chamada; -1 = pacote perdido (ok=false)
	call int
}

func (f *fakeRTTProber) PingRTT(ctx context.Context, dst netip.Addr, timeout time.Duration) (int, bool, error) {
	if f.call >= len(f.rtts) {
		return 0, false, nil
	}
	v := f.rtts[f.call]
	f.call++
	if v < 0 {
		return 0, false, nil
	}
	return v, true, nil
}

func TestMeasureLatencyFromRTTs(t *testing.T) {
	rtts := []int{40, 42, 38, 45, 41}
	rep, err := MeasureLatencyFromRTTs("8.8.8.8", rtts, 5)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if rep.MinRTT != 38 {
		t.Errorf("MinRTT = %d, esperado 38", rep.MinRTT)
	}
	if rep.MaxRTT != 45 {
		t.Errorf("MaxRTT = %d, esperado 45", rep.MaxRTT)
	}
	if rep.AvgRTT != 41 { // (40+42+38+45+41)/5 = 206/5 = 41.2 -> 41 (int division)
		t.Errorf("AvgRTT = %d, esperado 41", rep.AvgRTT)
	}
	if rep.PacketsLost != 0 {
		t.Errorf("PacketsLost = %d, esperado 0", rep.PacketsLost)
	}
	if rep.PacketsSent != 5 {
		t.Errorf("PacketsSent = %d, esperado 5", rep.PacketsSent)
	}
}

func TestMeasureLatencyFromRTTsWithLoss(t *testing.T) {
	rtts := []int{40, 42, 38} // 3 de 5 enviados — 2 perdidos
	rep, err := MeasureLatencyFromRTTs("8.8.8.8", rtts, 5)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if rep.PacketsLost != 2 {
		t.Errorf("PacketsLost = %d, esperado 2", rep.PacketsLost)
	}
}

func TestMeasureLatencyFromRTTsEmpty(t *testing.T) {
	rep, err := MeasureLatencyFromRTTs("8.8.8.8", nil, 10)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if rep.PacketsLost != 10 {
		t.Errorf("PacketsLost = %d, esperado 10 (todos perdidos)", rep.PacketsLost)
	}
	if rep.AvgRTT != 0 {
		t.Errorf("AvgRTT = %d, esperado 0 sem amostras", rep.AvgRTT)
	}
}

func TestMeasureLatencyFromRTTsDefaultHost(t *testing.T) {
	rep, err := MeasureLatencyFromRTTs("", []int{10}, 1)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if rep.Host != "8.8.8.8" {
		t.Errorf("Host = %q, esperado default 8.8.8.8", rep.Host)
	}
}

func TestCompareLatencyImproved(t *testing.T) {
	before := Benchmark{Latency: LatencyReport{AvgRTT: 100}, Jitter: 10}
	after := Benchmark{Latency: LatencyReport{AvgRTT: 80}, Jitter: 10}

	delta := Compare(before, after)
	if delta.LatencyAbsoluteDelta != -20 {
		t.Errorf("LatencyAbsoluteDelta = %d, esperado -20", delta.LatencyAbsoluteDelta)
	}
	if delta.LatencyDeltaPercent != -20 {
		t.Errorf("LatencyDeltaPercent = %.1f, esperado -20", delta.LatencyDeltaPercent)
	}
	if delta.Interpretation == "" {
		t.Error("Interpretation não deveria ser vazia")
	}
}

func TestCompareLatencyWorsened(t *testing.T) {
	before := Benchmark{Latency: LatencyReport{AvgRTT: 50}}
	after := Benchmark{Latency: LatencyReport{AvgRTT: 70}}

	delta := Compare(before, after)
	if delta.LatencyAbsoluteDelta != 20 {
		t.Errorf("LatencyAbsoluteDelta = %d, esperado 20", delta.LatencyAbsoluteDelta)
	}
	if delta.LatencyDeltaPercent <= 10 {
		t.Errorf("LatencyDeltaPercent = %.1f, esperado >10 (piora significativa)", delta.LatencyDeltaPercent)
	}
}

func TestCompareLatencyNoiseZone(t *testing.T) {
	// Diferença pequena (<5%) deve ser tratada como ruído, não melhoria real.
	before := Benchmark{Latency: LatencyReport{AvgRTT: 100}}
	after := Benchmark{Latency: LatencyReport{AvgRTT: 98}} // -2%

	delta := Compare(before, after)
	if delta.LatencyDeltaPercent >= 0 {
		t.Errorf("esperado delta negativo pequeno, obteve %.1f", delta.LatencyDeltaPercent)
	}
	// A interpretação deve mencionar que é ruído/pequeno demais
	if delta.Interpretation == "" {
		t.Error("Interpretation não deveria ser vazia")
	}
}

func TestMeasureLatencyWithFakeProber(t *testing.T) {
	fake := &fakeRTTProber{rtts: []int{40, 42, 38, 45, 41}}
	rep, err := measureLatency(context.Background(), fake, "8.8.8.8", 5)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if rep.PacketsSent != 5 || rep.PacketsLost != 0 {
		t.Errorf("sent=%d lost=%d, esperado sent=5 lost=0", rep.PacketsSent, rep.PacketsLost)
	}
	if rep.AvgRTT != 41 {
		t.Errorf("AvgRTT = %d, esperado 41", rep.AvgRTT)
	}
}

func TestMeasureLatencyWithPacketLoss(t *testing.T) {
	fake := &fakeRTTProber{rtts: []int{40, -1, 38, -1, 41}} // 2 perdidos de 5
	rep, err := measureLatency(context.Background(), fake, "8.8.8.8", 5)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if rep.PacketsLost != 2 {
		t.Errorf("PacketsLost = %d, esperado 2", rep.PacketsLost)
	}
	if rep.PacketsSent != 5 {
		t.Errorf("PacketsSent = %d, esperado 5", rep.PacketsSent)
	}
}

func TestMeasureLatencyInvalidHost(t *testing.T) {
	fake := &fakeRTTProber{rtts: []int{40}}
	_, err := measureLatency(context.Background(), fake, "esse.host.nao.existe.invalido.test", 1)
	if err == nil {
		t.Error("esperava erro para host não resolvível")
	}
}

func TestCompareZeroBefore(t *testing.T) {
	// Não deve dar panic com AvgRTT=0 (divisão por zero)
	before := Benchmark{Latency: LatencyReport{AvgRTT: 0}}
	after := Benchmark{Latency: LatencyReport{AvgRTT: 50}}

	delta := Compare(before, after)
	if delta.LatencyDeltaPercent != 0 {
		t.Errorf("LatencyDeltaPercent = %.1f, esperado 0 quando before=0", delta.LatencyDeltaPercent)
	}
}
