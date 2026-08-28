package netdiag

import (
	"context"
	"testing"
)

func TestProvedoresPadrao(t *testing.T) {
	prov := ProvedoresPadrao()
	if len(prov) < 4 {
		t.Errorf("esperado pelo menos 4 provedores padrão, obteve %d", len(prov))
	}

	for _, p := range prov {
		if p.Nome == "" || len(p.IPs) == 0 || p.DoHURL == "" {
			t.Errorf("provedor inválido: %+v", p)
		}
	}
}

func TestBenchmarkDNSMockHosts(t *testing.T) {
	// Com contexto já cancelado para testar ordenação e tratamento sem rede real
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	res := BenchmarkDNS(ctx, []string{"google.com"})
	if len(res) == 0 {
		t.Fatal("resultado do benchmark de DNS não pode ser vazio")
	}
}
