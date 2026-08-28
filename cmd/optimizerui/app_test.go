package main

import (
	"context"
	"path/filepath"
	"testing"

	"optimizer/internal/engine"
	"optimizer/internal/history"
	"optimizer/internal/tweaks"
	"optimizer/internal/winreg"
)

func novoAppTeste(t *testing.T) *App {
	t.Helper()
	f := winreg.NewFake()
	store, err := history.Open(filepath.Join(t.TempDir(), "test_history.jsonl"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	app := NewApp()
	app.ctx = context.Background()
	app.eng = &engine.Engine{
		Registry: tweaks.Build(f),
		History:  store,
		Elevated: false,
	}
	return app
}

func TestAppDiagnosticar(t *testing.T) {
	app := novoAppTeste(t)

	diagPessoal := app.Diagnosticar("pessoal")
	if diagPessoal.Total != 15 {
		t.Errorf("Total pessoal = %d, esperado 15", diagPessoal.Total)
	}
	if diagPessoal.Perfil != "pessoal" {
		t.Errorf("Perfil = %s, esperado pessoal", diagPessoal.Perfil)
	}

	diagTrabalho := app.Diagnosticar("trabalho")
	if diagTrabalho.Total != 14 {
		t.Errorf("Total trabalho = %d, esperado 14", diagTrabalho.Total)
	}
	if diagTrabalho.Perfil != "trabalho" {
		t.Errorf("Perfil = %s, esperado trabalho", diagTrabalho.Perfil)
	}
}

func TestAppAplicarSimular(t *testing.T) {
	app := novoAppTeste(t)

	// Sem IDs
	resVazio := app.Aplicar(nil, true, false)
	if resVazio != nil {
		t.Errorf("esperado nil para lista vazia de IDs")
	}

	// Simulação de item do usuário
	res := app.Aplicar([]string{"visual.menu-show-delay"}, true, false)
	if len(res) != 1 {
		t.Fatalf("esperado 1 resultado, obteve %d", len(res))
	}
	if res[0].Estado != "ok" {
		t.Errorf("estado = %s, esperado ok na simulação", res[0].Estado)
	}

	// Histórico deve conter a simulação
	hist := app.Historico()
	if len(hist) != 1 {
		t.Fatalf("esperado 1 entrada no histórico, obteve %d", len(hist))
	}
}

func TestAppDesfazerSemPendencias(t *testing.T) {
	app := novoAppTeste(t)

	res := app.DesfazerTudo()
	if len(res) != 0 {
		t.Errorf("esperado 0 resultados para desfazer sem pendências, obteve %d", len(res))
	}

	resIndiv := app.Desfazer([]string{"visual.menu-show-delay"})
	if len(resIndiv) != 1 || resIndiv[0].Estado != "pulado" {
		t.Errorf("esperado resultado 'pulado' para item sem histórico de apply")
	}
}

func TestAppListarEAplicarPerfisRede(t *testing.T) {
	app := novoAppTeste(t)

	perfis := app.ListarPerfisRede()
	if len(perfis) != 4 {
		t.Fatalf("esperado 4 perfis de rede, obteve %d", len(perfis))
	}

	// Aplicar perfil válido em simulação
	res := app.AplicarPerfilRede("network-fast", true)
	if len(res) != 2 {
		t.Fatalf("esperado 2 tweaks aplicados no perfil network-fast, obteve %d", len(res))
	}

	// Perfil inexistente
	resInvalido := app.AplicarPerfilRede("inexistente", true)
	if len(resInvalido) != 1 || resInvalido[0].Estado != "falhou" {
		t.Errorf("esperado falha para perfil inexistente")
	}
}

func TestAppRelatorioComparativo(t *testing.T) {
	app := novoAppTeste(t)

	antes := BenchmarkUI{Host: "8.8.8.8", AvgRTT: 100, MinRTT: 90, MaxRTT: 120, StdDev: 10, PacketsSent: 10}
	depois := BenchmarkUI{Host: "8.8.8.8", AvgRTT: 80, MinRTT: 70, MaxRTT: 95, StdDev: 8, PacketsSent: 10}

	comp := app.RelatorioComparativo(antes, depois)
	if comp.DeltaLatencia == "" || comp.Interpretacao == "" {
		t.Errorf("Relatório comparativo incompleto: %+v", comp)
	}
}
