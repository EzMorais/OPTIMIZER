package netdiag_test

import (
	"context"
	"testing"
	"time"

	"optimizer/internal/netdiag"
)

func TestFlushRede(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	relatorio := netdiag.ExecutarFlushRede(ctx)
	if relatorio.Mensagem == "" {
		t.Errorf("Relatório de flush vazio: %+v", relatorio)
	}
}

func TestMatrizPingJogos(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	regioes := netdiag.ObterRegioesPadrao()
	if len(regioes) < 3 {
		t.Errorf("Esperado pelo menos 3 regiões de teste, obteve %d", len(regioes))
	}

	res := netdiag.MedirMatrizJogos(ctx)
	if len(res) != len(regioes) {
		t.Errorf("MedirMatrizJogos retornou %d itens, esperado %d", len(res), len(regioes))
	}
}
