package netdiag

import (
	"context"
	"net"
	"sort"
	"sync"
	"time"
)

// DNSProvider representa um servidor DNS público e seguro com base no GRC Benchmark.
type DNSProvider struct {
	Nome        string   `json:"nome"`
	IPs         []string `json:"ips"`
	DoHURL      string   `json:"dohUrl"`
	Privacidade string   `json:"privacidade"` // "Sem Logs", "Segurança/Malware", "Padrão", etc.
	AvgRTT      int      `json:"avgRttMs"`
	Perda       float64  `json:"perda"`
	Recomendado bool     `json:"recomendado"`
}

// ProvedoresPadrao lista a base completa de resolvedores DNS globais e regionais do GRC Benchmark.
func ProvedoresPadrao() []DNSProvider {
	return []DNSProvider{
		{
			Nome:        "Cloudflare (1.1.1.1)",
			IPs:         []string{"1.1.1.1", "1.0.0.1"},
			DoHURL:      "https://cloudflare-dns.com/dns-query",
			Privacidade: "Sem Logs / Ultra Baixa Latência",
		},
		{
			Nome:        "Cloudflare Security (1.1.1.2)",
			IPs:         []string{"1.1.1.2", "1.0.0.2"},
			DoHURL:      "https://security.cloudflare-dns.com/dns-query",
			Privacidade: "Bloqueio Automático de Malware & Phishing",
		},
		{
			Nome:        "Google Public DNS (8.8.8.8)",
			IPs:         []string{"8.8.8.8", "8.8.4.4"},
			DoHURL:      "https://dns.google/dns-query",
			Privacidade: "Alta Disponibilidade Global (Anycast)",
		},
		{
			Nome:        "Quad9 Security (9.9.9.9)",
			IPs:         []string{"9.9.9.9", "149.112.112.112"},
			DoHURL:      "https://dns.quad9.net/dns-query",
			Privacidade: "Bloqueio de Ameaças & Sem Logs (Suíça)",
		},
		{
			Nome:        "OpenDNS / Cisco (208.67.222.222)",
			IPs:         []string{"208.67.222.222", "208.67.220.220"},
			DoHURL:      "https://doh.opendns.com/dns-query",
			Privacidade: "Filtro de Segurança Cisco Umbrella",
		},
		{
			Nome:        "Level 3 / Lumen (4.2.2.1)",
			IPs:         []string{"4.2.2.1", "4.2.2.2"},
			DoHURL:      "https://dns.level3.net/dns-query",
			Privacidade: "Backbone Global Tier-1 (GRC Classic)",
		},
		{
			Nome:        "AdGuard DNS (94.140.14.14)",
			IPs:         []string{"94.140.14.14", "94.140.15.15"},
			DoHURL:      "https://dns.adguard-dns.com/dns-query",
			Privacidade: "Bloqueio de Anúncios e Rastreadores",
		},
		{
			Nome:        "Control D (76.76.2.0)",
			IPs:         []string{"76.76.2.0", "76.76.10.0"},
			DoHURL:      "https://freedns.controld.com/p0",
			Privacidade: "Sem Logs & Resposta de Alta Velocidade",
		},
		{
			Nome:        "CleanBrowsing (185.228.168.9)",
			IPs:         []string{"185.228.168.9", "185.228.169.9"},
			DoHURL:      "https://doh.cleanbrowsing.org/doh/security-filter/",
			Privacidade: "Filtro de Segurança e Bloqueio Malicioso",
		},
		{
			Nome:        "Comodo / Dyn DNS (8.26.56.26)",
			IPs:         []string{"8.26.56.26", "8.20.247.20"},
			DoHURL:      "https://dns.comodo.com/dns-query",
			Privacidade: "Segurança de Redirecionamento",
		},
		{
			Nome:        "GigaDNS Brasil (189.38.95.95)",
			IPs:         []string{"189.38.95.95", "189.38.95.96"},
			DoHURL:      "https://doh.gigadns.com.br/dns-query",
			Privacidade: "Roteamento Otimizado para o Brasil",
		},
		{
			Nome:        "Verisign Public DNS (64.6.64.6)",
			IPs:         []string{"64.6.64.6", "64.6.65.6"},
			DoHURL:      "https://doh.verisign.com/dns-query",
			Privacidade: "Estabilidade e Sem Venda de Dados",
		},
	}
}

// BenchmarkDNS mede a velocidade de resolução de cada provedor DNS de forma concorrente.
func BenchmarkDNS(ctx context.Context, hostsTeste []string) []DNSProvider {
	if len(hostsTeste) == 0 {
		hostsTeste = []string{"microsoft.com", "google.com", "github.com", "cloudflare.com"}
	}

	provedores := ProvedoresPadrao()
	var wg sync.WaitGroup

	for i := range provedores {
		wg.Add(1)
		go func(p *DNSProvider) {
			defer wg.Done()
			server := p.IPs[0] + ":53"

			var tempos []int
			for _, host := range hostsTeste {
				d := net.Dialer{Timeout: 700 * time.Millisecond}
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

				subCtx, cancel := context.WithTimeout(ctx, 800*time.Millisecond)
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
		}(&provedores[i])
	}

	wg.Wait()

	// Ordena do mais rápido ao mais lento
	sort.Slice(provedores, func(i, j int) bool {
		if provedores[i].Perda == provedores[j].Perda {
			return provedores[i].AvgRTT < provedores[j].AvgRTT
		}
		return provedores[i].Perda < provedores[j].Perda
	})

	// Marca o melhor como recomendado se responder com sucesso
	if len(provedores) > 0 && provedores[0].Perda < 100 {
		provedores[0].Recomendado = true
	}

	return provedores
}
