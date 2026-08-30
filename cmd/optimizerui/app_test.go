package main

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"optimizer/internal/engine"
	"optimizer/internal/history"
	"optimizer/internal/telemetry"
	"optimizer/internal/tweaks"
	"optimizer/internal/winreg"
)

type fakeAppDNSRunner struct {
	outputs [][]byte
	scripts []string
}

func (f *fakeAppDNSRunner) RunPowerShell(_ context.Context, script string) ([]byte, error) {
	f.scripts = append(f.scripts, script)
	if len(f.outputs) == 0 {
		return nil, nil
	}
	out := f.outputs[0]
	f.outputs = f.outputs[1:]
	return out, nil
}

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
	if diagPessoal.Total != 55 {
		t.Errorf("Total pessoal = %d, esperado 55", diagPessoal.Total)
	}
	if diagPessoal.Perfil != "pessoal" {
		t.Errorf("Perfil = %s, esperado pessoal", diagPessoal.Perfil)
	}

	diagTrabalho := app.Diagnosticar("trabalho")
	if diagTrabalho.Total != 40 {
		t.Errorf("Total trabalho = %d, esperado 40", diagTrabalho.Total)
	}
	if diagTrabalho.Perfil != "trabalho" {
		t.Errorf("Perfil = %s, esperado trabalho", diagTrabalho.Perfil)
	}
}

func TestAppRecomendacoesRespeitamGPUDetectada(t *testing.T) {
	app := novoAppTeste(t)
	app.benchCollector = telemetry.NewCollector(&telemetry.MockProvider{HardwareInfo: telemetry.HardwareStaticInfo{
		CPUName: "CPU teste", PhysicalCores: 4, LogicalCores: 8, GPUName: "GPU integrada", TotalRAMMB: 8192,
	}})
	diag := app.Diagnosticar("pessoal")
	for _, item := range diag.Itens {
		if strings.Contains(item.ID, "nvidia") && item.Recomendado {
			t.Fatalf("%s foi recomendado sem GPU NVIDIA: %s", item.ID, item.MotivoRecomendacao)
		}
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

	// Aplicar perfil NVIDIA em simulação
	resNvidia := app.AplicarPerfilUso("nvidia", true)
	if len(resNvidia) == 0 {
		t.Fatal("esperado resultado para perfil nvidia")
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

func TestAppResumoVisaoMostraCoberturaECategoriasDoCatalogo(t *testing.T) {
	app := novoAppTeste(t)
	antes := app.ResumoVisao("pessoal")

	// Uma aplicação real no registro falso torna a cobertura observável para a UI.
	res := app.Aplicar([]string{"visual.menu-show-delay"}, false, false)
	if len(res) != 1 || res[0].Estado != "ok" {
		t.Fatalf("não foi possível preparar o ajuste aplicado: %+v", res)
	}

	visao := app.ResumoVisao("pessoal")
	if visao.TotalAjustes != 55 {
		t.Errorf("total de ajustes = %d, esperado 55", visao.TotalAjustes)
	}
	if visao.Aplicados != antes.Aplicados+1 {
		t.Errorf("aplicados = %d, esperado %d após aplicar um ajuste", visao.Aplicados, antes.Aplicados+1)
	}
	if visao.CoberturaPercentual <= antes.CoberturaPercentual {
		t.Errorf("cobertura = %.2f%%, esperado valor maior que %.2f%%", visao.CoberturaPercentual, antes.CoberturaPercentual)
	}

	totalCategorias := 0
	for _, categoria := range visao.Categorias {
		totalCategorias += categoria.Total
	}
	if totalCategorias != visao.TotalAjustes {
		t.Errorf("categorias somam %d ajustes, esperado %d", totalCategorias, visao.TotalAjustes)
	}
}

func TestAppMostraEAplicaDNSDaInterfaceAtiva(t *testing.T) {
	app := novoAppTeste(t)
	app.eng.Elevated = true
	runner := &fakeAppDNSRunner{outputs: [][]byte{
		[]byte(`{"interfaceAlias":"Ethernet","interfaceIndex":12,"serverAddresses":["192.168.1.1"]}`),
		[]byte(`{"interfaceAlias":"Ethernet","interfaceIndex":12,"serverAddresses":["192.168.1.1"]}`),
	}}
	app.dnsRunner = runner

	atual := app.ObterDNSAtual()
	if atual.Erro != "" || atual.Interface != "Ethernet" {
		t.Fatalf("DNS atual inesperado: %+v", atual)
	}
	if len(atual.Servidores) != 1 || atual.Servidores[0] != "192.168.1.1" {
		t.Fatalf("servidores atuais inesperados: %+v", atual.Servidores)
	}

	resultado := app.AplicarDNS([]string{"1.1.1.1", "1.0.0.1"})
	if !resultado.Ok {
		t.Fatalf("AplicarDNS falhou: %+v", resultado)
	}
	if len(runner.scripts) != 3 || !strings.Contains(runner.scripts[2], "Set-DnsClientServerAddress") {
		t.Fatalf("configuração de DNS não foi executada: %+v", runner.scripts)
	}
}

func TestAppTelemetriaAoVivo(t *testing.T) {
	app := novoAppTeste(t)
	mock := &telemetry.MockProvider{
		Samples: []telemetry.MetricSample{
			{
				CPUUsagePercent:   25.5,
				CPUFrequencyMHz:   3600,
				GPUUsagePercent:   42.0,
				GPUMemoryUsedMB:   2048,
				RAMUsedMB:         8192,
				RAMTotalMB:        32768,
				ThermalThrottling: false,
			},
		},
		HardwareInfo: telemetry.HardwareStaticInfo{
			PhysicalCores: 8,
			LogicalCores:  16,
			TotalRAMMB:    32768,
			TotalGPUMemMB: 8192,
		},
	}
	app.benchCollector = telemetry.NewCollector(mock)

	telem := app.ObterTelemetriaAoVivo()
	if telem.CPUUsagePercent != 25.5 {
		t.Errorf("CPUUsagePercent = %f, esperado 25.5", telem.CPUUsagePercent)
	}
	if telem.GPUUsagePercent != 42.0 {
		t.Errorf("GPUUsagePercent = %f, esperado 42.0", telem.GPUUsagePercent)
	}
	if telem.RAMUsedMB != 8192 {
		t.Errorf("RAMUsedMB = %f, esperado 8192", telem.RAMUsedMB)
	}
	if telem.PhysicalCores != 8 || telem.LogicalProcessors != 16 {
		t.Errorf("Cores = %d/%d, esperado 8/16", telem.PhysicalCores, telem.LogicalProcessors)
	}
}

func TestAppExportarRelatorioSistema(t *testing.T) {
	app := novoAppTeste(t)
	relatorio := app.ExportarRelatorioSistema()
	if !strings.Contains(relatorio, "Relatório Completo de Diagnóstico") {
		t.Errorf("Relatório gerado inválido:\n%s", relatorio)
	}
	if !strings.Contains(relatorio, "Total de Ajustes") {
		t.Errorf("Relatório deve conter contagem de ajustes:\n%s", relatorio)
	}
}

func TestAppFlushingRede(t *testing.T) {
	app := novoAppTeste(t)
	res := app.FlushingRede()
	if res.Mensagem == "" {
		t.Errorf("FlushingRede retornou mensagem vazia: %+v", res)
	}
}

func TestAppMatrizPingJogos(t *testing.T) {
	app := novoAppTeste(t)
	res := app.MatrizPingJogos()
	if len(res) == 0 {
		t.Errorf("MatrizPingJogos retornou 0 regiões: %+v", res)
	}
}

func TestAppMedirRedeComPacotes(t *testing.T) {
	app := novoAppTeste(t)
	res := app.MedirRedeComPacotes("8.8.8.8", 3)
	if res.Host != "8.8.8.8" {
		t.Errorf("Host incorreto: %s", res.Host)
	}
}

func TestAppMTU(t *testing.T) {
	app := novoAppTeste(t)
	res := app.MedirMTU("8.8.8.8")
	if res.Erro != "" && res.MTUAtual == 0 {
		// Normal em ambiente restrito ou sem IPv4 ativo
		return
	}
	if res.MTUAtual > 0 && res.MTUAtual < 576 {
		t.Errorf("MTU inválido: %d", res.MTUAtual)
	}
}

func TestAppTimerResolution(t *testing.T) {
	app := novoAppTeste(t)
	info := app.ObterTimerResolution()
	if info.MinResolutionMs <= 0 || info.MaxResolutionMs <= 0 {
		t.Errorf("Timer resolution inválida: %+v", info)
	}

	setRes := app.DefinirTimerResolution(0.5, true)
	if !setRes.OK && setRes.Mensagem == "" {
		t.Errorf("Resposta de DefinirTimerResolution inválida: %+v", setRes)
	}
}

func TestAppMedirSleepPrecision(t *testing.T) {
	app := novoAppTeste(t)
	res := app.MedirSleepPrecision(10)
	if len(res.Samples) != 10 || res.AverageMs <= 0 {
		t.Errorf("Sleep precision retornou dados inválidos: %+v", res)
	}
}

func TestAppDispositivosMSI(t *testing.T) {
	app := novoAppTeste(t)
	devs := app.ListarDispositivosMSI()
	if devs == nil {
		t.Fatal("ListarDispositivosMSI não deve retornar nil")
	}

	res := app.AlternarModoMSI("SYSTEM\\Test", true)
	if res.Mensagem == "" {
		t.Errorf("Mensagem vazia em AlternarModoMSI: %+v", res)
	}
}
