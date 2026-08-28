package netdiag

import (
	"context"
	"sync"
	"time"
)

// RegiaoPing representa uma região de servidores de jogos para medição de latência.
type RegiaoPing struct {
	Codigo      string  `json:"codigo"`
	Nome        string  `json:"nome"`
	Localizacao string  `json:"localizacao"`
	Bandeira    string  `json:"bandeira"`
	Host        string  `json:"host"`
	PingMS      int     `json:"pingMs"`
	JitterMS    float64 `json:"jitterMs"`
	Status      string  `json:"status"` // "otimo" (<40ms), "bom" (40-90ms), "regular" (90-150ms), "alto" (>150ms), "indisponivel"
}

// ObterRegioesPadrao devolve a lista de nós globais estratégicos de jogos.
func ObterRegioesPadrao() []RegiaoPing {
	return []RegiaoPing{
		{
			Codigo:      "br-sp",
			Nome:        "Brasil (São Paulo)",
			Localizacao: "América do Sul (Principal)",
			Bandeira:    "🇧🇷",
			Host:        "189.38.95.95", // CDN / Game Node BR
		},
		{
			Codigo:      "cl-stg",
			Nome:        "Chile (Santiago)",
			Localizacao: "América do Sul (Pacífico)",
			Bandeira:    "🇨🇱",
			Host:        "200.73.14.3",
		},
		{
			Codigo:      "us-east",
			Nome:        "EUA Leste (Virgínia)",
			Localizacao: "América do Norte",
			Bandeira:    "🇺🇸",
			Host:        "8.8.8.8",
		},
		{
			Codigo:      "eu-fra",
			Nome:        "Europa (Frankfurt)",
			Localizacao: "Europa Central",
			Bandeira:    "🇩🇪",
			Host:        "1.1.1.1",
		},
	}
}

// MedirMatrizJogos realiza ping simultâneo não-bloqueante para todas as regiões.
func MedirMatrizJogos(ctx context.Context) []RegiaoPing {
	regioes := ObterRegioesPadrao()
	var wg sync.WaitGroup
	resultados := make([]RegiaoPing, len(regioes))

	for i, reg := range regioes {
		wg.Add(1)
		go func(idx int, r RegiaoPing) {
			defer wg.Done()
			timeoutCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
			defer cancel()

			bench, _ := MeasureLatency(timeoutCtx, r.Host, 4)
			r.PingMS = bench.AvgRTT
			r.JitterMS = float64(bench.StdDev)

			if bench.AvgRTT <= 0 || bench.PacketsLost == 4 {
				r.Status = "indisponivel"
			} else if bench.AvgRTT <= 35 {
				r.Status = "otimo"
			} else if bench.AvgRTT <= 85 {
				r.Status = "bom"
			} else if bench.AvgRTT <= 140 {
				r.Status = "regular"
			} else {
				r.Status = "alto"
			}
			resultados[idx] = r
		}(i, reg)
	}

	wg.Wait()
	return resultados
}
