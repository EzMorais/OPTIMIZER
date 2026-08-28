package netdiag

import (
	"context"
	"net"
	"sort"
	"time"
)

// DNSProvider representa um servidor DNS público conhecido.
type DNSProvider struct {
	Nome        string   `json:"nome"`
	IPs         []string `json:"ips"`
	DoHURL      string   `json:"dohUrl"`
	Privacidade string   `json:"privacidade"` // "Sem Logs", "Segurança/Malware", "Padrão"
	AvgRTT      int      `json:"avgRttMs"`
	Perda       float64  `json:"perda"`
	Recomendado bool     `json:"recomendado"`
}

// ProvedoresPadrao lista os principais resolvedores DNS globais e seguros.
func ProvedoresPadrao() []DNSProvider {
	return []DNSProvider{
		{
			Nome:        "Cloudflare (1.1.1.1)",
			IPs:         []string{"1.1.1.1", "1.0.0.1"},
			DoHURL:      "https://cloudflare-dns.com/dns-query",
			Privacidade: "Sem Logs / Foco em Velocidade",
		},
		{
			Nome:        "Google Public DNS (8.8.8.8)",
			IPs:         []string{"8.8.8.8", "8.8.4.4"},
			DoHURL:      "https://dns.google/dns-query",
			Privacidade: "Alta Disponibilidade Global",
		},
		{
			Nome:        "Quad9 (9.9.9.9)",
			IPs:         []string{"9.9.9.9", "149.112.112.112"},
			DoHURL:      "https://dns.quad9.net/dns-query",
			Privacidade: "Bloqueio Nativo de Ameaças / Malware",
		},
		{
			Nome:        "OpenDNS (Cisco)",
			IPs:         []string{"208.67.222.222", "208.67.220.220"},
			DoHURL:      "https://doh.opendns.com/dns-query",
			Privacidade: "Filtro de Segurança e Família",
		},
	}
}

// BenchmarkDNS mede a velocidade de resolução de cada provedor DNS.
func BenchmarkDNS(ctx context.Context, hostsTeste []string) []DNSProvider {
	if len(hostsTeste) == 0 {
		hostsTeste = []string{"microsoft.com", "google.com", "github.com", "cloudflare.com"}
	}

	provedores := ProvedoresPadrao()
	for i := range provedores {
		p := &provedores[i]
		server := p.IPs[0] + ":53"

		var tempos []int
		for _, host := range hostsTeste {
			d := net.Dialer{Timeout: 800 * time.Millisecond}
			inicio := time.Now()

			conn, err := d.DialContext(ctx, "udp", server)
			if err != nil {
				continue
			}
			_ = conn.Close()

			r := &net.Resolver{
				PreferGo: true,
				Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
					return d.DialContext(ctx, "udp", server)
				},
			}

			subCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
			_, err = r.LookupHost(subCtx, host)
			cancel()

			dur := int(time.Since(inicio).Milliseconds())
			if err == nil {
				tempos = append(tempos, dur)
			}
		}

		if len(tempos) > 0 {
			soma := 0
			for _, t := range tempos {
				soma += t
			}
			p.AvgRTT = soma / len(tempos)
			p.Perda = float64(len(hostsTeste)-len(tempos)) / float64(len(hostsTeste)) * 100
		} else {
			p.AvgRTT = 999
			p.Perda = 100
		}
	}

	// Ordena pelo menor RTT
	sort.Slice(provedores, func(i, j int) bool {
		return provedores[i].AvgRTT < provedores[j].AvgRTT
	})

	if len(provedores) > 0 && provedores[0].AvgRTT < 900 {
		provedores[0].Recomendado = true
	}

	return provedores
}
