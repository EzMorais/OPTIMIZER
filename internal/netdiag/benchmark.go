// Package netdiag fornece medição de rede (ping, latência, jitter, throughput).
package netdiag

import (
	"context"
	"fmt"
	"math"
	"time"
)

// LatencyReport contém estatísticas de latência de um host.
type LatencyReport struct {
	Host       string `json:"host"`
	MinRTT     int    `json:"min_rtt_ms"`
	AvgRTT     int    `json:"avg_rtt_ms"`
	MaxRTT     int    `json:"max_rtt_ms"`
	StdDev     int    `json:"stddev_ms"`
	PacketsLost int    `json:"packets_lost"`
	PacketsSent int    `json:"packets_sent"`
}

// Benchmark contém todas as métricas de desempenho de rede capturadas antes/depois de um ajuste.
type Benchmark struct {
	Timestamp   time.Time     `json:"timestamp"`
	Host        string        `json:"host"`
	Latency     LatencyReport `json:"latency"`
	Jitter      int           `json:"jitter_ms"` // stddev da latência
	Loss        float64       `json:"loss_percent"`
	CPUUsage    float64       `json:"cpu_usage_percent,omitempty"`
	NetworkBW   float64       `json:"network_bw_mbps,omitempty"`
	TCPErrors   uint64        `json:"tcp_errors,omitempty"`
}

// BenchmarkDelta compara dois benchmarks e computa a diferença.
type BenchmarkDelta struct {
	Before              Benchmark  `json:"before"`
	After               Benchmark  `json:"after"`
	LatencyDeltaPercent float64    `json:"latency_delta_percent"` // (after.AvgRTT - before.AvgRTT) / before.AvgRTT * 100
	LatencyAbsoluteDelta int       `json:"latency_absolute_delta_ms"`
	JitterDeltaPercent  float64    `json:"jitter_delta_percent"`
	LossDeltaPercent    float64    `json:"loss_delta_percent"`
	Interpretation      string    `json:"interpretation"`
}

// MeasureLatency executa um benchmark de latência contra um host, usando RTTs fornecidos.
// Computa estatísticas a partir de uma lista de RTTs em ms.
func MeasureLatency(host string, rtts []int, packetsSent int) (LatencyReport, error) {
	if host == "" {
		host = "8.8.8.8"
	}
	if len(rtts) == 0 {
		return LatencyReport{
			Host:        host,
			PacketsSent: packetsSent,
			PacketsLost: packetsSent,
		}, nil
	}

	min, max := rtts[0], rtts[0]
	sum := int64(0)
	for _, rtt := range rtts {
		if rtt < min {
			min = rtt
		}
		if rtt > max {
			max = rtt
		}
		sum += int64(rtt)
	}

	avg := sum / int64(len(rtts))

	// Calcular desvio padrão (jitter).
	sumSq := int64(0)
	for _, rtt := range rtts {
		delta := int64(rtt) - avg
		sumSq += delta * delta
	}
	variance := sumSq / int64(len(rtts))
	stdDev := int(math.Sqrt(float64(variance)))

	lost := packetsSent - len(rtts)

	return LatencyReport{
		Host:        host,
		MinRTT:      int(min),
		AvgRTT:      int(avg),
		MaxRTT:      int(max),
		StdDev:      stdDev,
		PacketsLost: lost,
		PacketsSent: packetsSent,
	}, nil
}

// MakeBenchmark cria um snapshot de benchmark no momento presente,
// usando latência medida e opcionalmente performance counters.
func MakeBenchmark(ctx context.Context, latency LatencyReport, host string) Benchmark {
	return Benchmark{
		Timestamp:   time.Now(),
		Host:        host,
		Latency:     latency,
		Jitter:      latency.StdDev,
		Loss:        float64(latency.PacketsLost) / float64(latency.PacketsSent) * 100,
		CPUUsage:    0, // TODO: coletar via WMI Win32_PerfFormattedData_PerfOS_Processor
		NetworkBW:   0, // TODO: coletar via WMI Win32_PerfFormattedData_Tcpip_NetworkInterface
		TCPErrors:   0, // TODO: coletar via WMI Win32_PerfFormattedData_Tcpip_TCPv4
	}
}

// Compare calcula as diferenças entre dois benchmarks.
func Compare(before, after Benchmark) BenchmarkDelta {
	delta := BenchmarkDelta{
		Before: before,
		After:  after,
	}

	if before.Latency.AvgRTT > 0 {
		latencyAbs := after.Latency.AvgRTT - before.Latency.AvgRTT
		delta.LatencyAbsoluteDelta = latencyAbs
		delta.LatencyDeltaPercent = float64(latencyAbs) / float64(before.Latency.AvgRTT) * 100
	}

	if before.Jitter > 0 {
		delta.JitterDeltaPercent = float64(after.Jitter-before.Jitter) / float64(before.Jitter) * 100
	}

	delta.LossDeltaPercent = after.Loss - before.Loss

	// Interpretação honesta do resultado.
	delta.Interpretation = interpretBenchmarkDelta(delta)

	return delta
}

func interpretBenchmarkDelta(delta BenchmarkDelta) string {
	// Se latência piorou significativamente (>10%), avisar.
	if delta.LatencyDeltaPercent > 10 {
		return fmt.Sprintf(
			"A latência aumentou %.1f%% (%d ms). "+
				"O ajuste pode ter piorado a conexão — considere desfazer.",
			delta.LatencyDeltaPercent, delta.LatencyAbsoluteDelta,
		)
	}

	// Se latência melhorou, mas margem < 5%, avisar que é ruído de medição.
	if delta.LatencyDeltaPercent > 0 && delta.LatencyDeltaPercent < 5 {
		return fmt.Sprintf(
			"A latência diminuiu %.1f%% (%d ms), mas esta mudança é pequena demais para ser "+
				"distinguível do ruído de medição — não há evidência de melhoria real.",
			-delta.LatencyDeltaPercent, -delta.LatencyAbsoluteDelta,
		)
	}

	// Se latência melhorou significativamente (>5%).
	if delta.LatencyDeltaPercent < -5 {
		return fmt.Sprintf(
			"A latência diminuiu %.1f%% (%d ms). Uma melhoria detectada.",
			-delta.LatencyDeltaPercent, -delta.LatencyAbsoluteDelta,
		)
	}

	// Se latência está estável.
	if delta.LatencyDeltaPercent > -5 && delta.LatencyDeltaPercent < 5 {
		if delta.LossDeltaPercent < 0 {
			return fmt.Sprintf(
				"A latência está estável (%.1f ms). Perda de pacotes diminuiu %.1f%%.",
				float64(delta.After.Latency.AvgRTT), -delta.LossDeltaPercent,
			)
		}
		return fmt.Sprintf(
			"A latência está estável (%.1f ms). Sem mudança perceptível.",
			float64(delta.After.Latency.AvgRTT),
		)
	}

	return "Benchmarks capturados."
}
