package main

import (
	"context"
	"path/filepath"
	"testing"

	"optimizer/internal/engine"
	"optimizer/internal/history"
	"optimizer/internal/telemetry"
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
	if diagPessoal.Total != 41 {
		t.Errorf("Total pessoal = %d, esperado 41", diagPessoal.Total)
	}
	if diagPessoal.Perfil != "pessoal" {
		t.Errorf("Perfil = %s, esperado pessoal", diagPessoal.Perfil)
	}

	diagTrabalho := app.Diagnosticar("trabalho")
	if diagTrabalho.Total != 29 {
		t.Errorf("Total trabalho = %d, esperado 29", diagTrabalho.Total)
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

func TestAppPerfisUso(t *testing.T) {
	app := novoAppTeste(t)

	perfis := app.ListarPerfisUso()
	if len(perfis) < 2 {
		t.Fatalf("esperado pelo menos 2 perfis de uso, obteve %d", len(perfis))
	}

	// Sem perfil ativo inicial
	ativo := app.ObterPerfilAtivo()
	if ativo != "" {
		t.Errorf("nenhum perfil deveria estar ativo inicialmente, obteve %q", ativo)
	}

	// Aplicar perfil JOGO em simulação
	resJogo := app.AplicarPerfilUso("jogo", true)
	if len(resJogo) == 0 {
		t.Fatal("esperado resultado para perfil jogo")
	}

	// Aplicar perfil CODING em simulação
	resCoding := app.AplicarPerfilUso("coding", true)
	if len(resCoding) == 0 {
		t.Fatal("esperado resultado para perfil coding")
	}

	// Perfil inválido
	resInvalido := app.AplicarPerfilUso("invalido", true)
	if len(resInvalido) != 1 || resInvalido[0].Estado != "falhou" {
		t.Errorf("esperado erro ao aplicar perfil inválido")
	}
}

func TestAppDispositivosFantasmas(t *testing.T) {
	app := novoAppTeste(t)
	devs := app.ListarDispositivosFantasmas()
	// No teste em fake/mock não deve quebrar
	if devs == nil {
		t.Errorf("esperado array não nulo de dispositivos")
	}

	res := app.LimparDispositivosFantasmas(nil)
	if res.Removidos != 0 {
		t.Errorf("esperado 0 removidos para lista vazia")
	}
}

func TestAppBenchmarkFlow(t *testing.T) {
	app := novoAppTeste(t)
	// Configura mock collector
	temp := 48.0
	mock := &telemetry.MockProvider{
		Samples: []telemetry.MetricSample{
			{
				CPUUsagePercent: 18.0,
				RAMUsedMB:       4000,
				RAMTotalMB:      16384,
				GPUUsagePercent: 25.0,
				CPUTempCelsius:  &temp,
			},
		},
	}
	app.benchCollector = telemetry.NewCollector(mock)

	// 1. Iniciar benchmark pré
	reportAntes, err := app.IniciarBenchmarkBase("jogo", 1)
	if err != nil {
		t.Fatalf("erro ao iniciar benchmark base: %v", err)
	}
	if reportAntes.SampleCount == 0 {
		t.Error("esperado pelo menos 1 amostra no benchmark prévio")
	}

	// 2. Aplicar perfil com benchmark prévio
	resApp := app.AplicarPerfilComBenchmark("jogo", true, reportAntes)
	if len(resApp.Resultados) == 0 {
		t.Error("esperado resultados ao aplicar perfil com benchmark")
	}
	if resApp.BatchID == "" {
		t.Error("esperado BatchID gerado para o lote de perfil")
	}

	// 3. Iniciar benchmark pós
	comp, err := app.IniciarBenchmarkPos("jogo", resApp.BatchID, 1)
	if err != nil {
		t.Fatalf("erro ao iniciar benchmark pos: %v", err)
	}
	if comp.ProfileKey != "jogo" {
		t.Errorf("esperado ProfileKey = jogo, obteve %s", comp.ProfileKey)
	}

	// 4. Cancelar benchmark
	app.CancelarBenchmark()
}

