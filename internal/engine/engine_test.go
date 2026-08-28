package engine_test

import (
	"context"
	"path/filepath"
	"testing"

	"optimizer/internal/engine"
	"optimizer/internal/history"
	"optimizer/internal/tweak"
	"optimizer/internal/tweaks"
	"optimizer/internal/winreg"
)

func novoMotor(t *testing.T, f *winreg.Fake, elevado bool) *engine.Engine {
	t.Helper()
	store, err := history.Open(filepath.Join(t.TempDir(), "history.jsonl"))
	if err != nil {
		t.Fatalf("abrindo histórico: %v", err)
	}
	return &engine.Engine{
		Registry: tweaks.Build(f),
		History:  store,
		Elevated: elevado,
	}
}

const idUsuario = "jogos.gamedvr-usuario"
const idMaquina = "rede.sem-upload-de-atualizacoes"

func TestAplicarEDesfazerVoltaAoEstadoOriginal(t *testing.T) {
	f := winreg.NewFake()
	f.Seed(winreg.HKCU, `System\GameConfigStore`, "GameDVR_Enabled", uint32(1))
	eng := novoMotor(t, f, false)
	ctx := context.Background()

	res := eng.Apply(ctx, []string{idUsuario}, false)
	if len(res) != 1 || res[0].Err != nil || !res[0].Applied {
		t.Fatalf("apply: %+v", res)
	}
	if got, _ := f.GetDWord(winreg.HKCU, `System\GameConfigStore`, "GameDVR_Enabled"); got != 0 {
		t.Fatalf("valor depois do apply = %d, queria 0", got)
	}

	res = eng.Revert(ctx, []string{idUsuario}, false)
	if len(res) != 1 || res[0].Err != nil {
		t.Fatalf("revert: %+v", res)
	}
	if got, _ := f.GetDWord(winreg.HKCU, `System\GameConfigStore`, "GameDVR_Enabled"); got != 1 {
		t.Fatalf("valor depois do revert = %d, queria 1", got)
	}
}

func TestAplicarDuasVezesNaoRefazTrabalho(t *testing.T) {
	f := winreg.NewFake()
	eng := novoMotor(t, f, false)
	ctx := context.Background()

	eng.Apply(ctx, []string{idUsuario}, false)
	res := eng.Apply(ctx, []string{idUsuario}, false)
	if !res[0].Skipped {
		t.Fatalf("segunda aplicação deveria ser pulada: %+v", res[0])
	}
}

func TestItemDeMaquinaEPuladoComExplicacaoSemAdministrador(t *testing.T) {
	f := winreg.NewFake()
	f.DenyPrefix("HKLM\\")
	eng := novoMotor(t, f, false)

	res := eng.Apply(context.Background(), []string{idMaquina}, false)
	if !res[0].Skipped {
		t.Fatalf("sem administrador, o item tinha que ser pulado: %+v", res[0])
	}
	if res[0].Reason == "" {
		t.Fatal("pulou sem explicar o porquê para o usuário")
	}
	if len(f.Keys()) != 0 {
		t.Fatalf("tentou gravar mesmo sem permissão: %v", f.Keys())
	}
}

func TestDesfazerSemHistoricoNaoAdivinhaValor(t *testing.T) {
	f := winreg.NewFake()
	eng := novoMotor(t, f, false)

	res := eng.Revert(context.Background(), []string{idUsuario}, false)
	if !res[0].Skipped {
		t.Fatalf("sem registro de aplicação não há o que desfazer: %+v", res[0])
	}
	if len(f.Keys()) != 0 {
		t.Fatal("o motor escreveu no registro para 'desfazer' algo que nunca aplicou")
	}
}

func TestSimulacaoNaoTocaNoRegistroNemCriaPendencia(t *testing.T) {
	f := winreg.NewFake()
	eng := novoMotor(t, f, false)
	ctx := context.Background()

	eng.Apply(ctx, []string{idUsuario}, true)
	if len(f.Keys()) != 0 {
		t.Fatalf("a simulação gravou: %v", f.Keys())
	}
	pend, err := eng.PendingIDs()
	if err != nil {
		t.Fatalf("PendingIDs: %v", err)
	}
	if len(pend) != 0 {
		t.Fatalf("a simulação criou pendência de desfazer: %v", pend)
	}
}

func TestDiagnosticoNaoAlteraNada(t *testing.T) {
	f := winreg.NewFake()
	f.Seed(winreg.HKCU, `System\GameConfigStore`, "GameDVR_Enabled", uint32(1))
	eng := novoMotor(t, f, false)

	antes := len(f.Keys())
	status := eng.Scan(context.Background(), tweak.ProfilePersonal)
	if len(status) == 0 {
		t.Fatal("diagnóstico vazio")
	}
	if len(f.Keys()) != antes {
		t.Fatal("o diagnóstico alterou o registro")
	}
}

func TestRecomendadosSaoPoucosEIntencionais(t *testing.T) {
	// Guarda contra o padrão escuro da categoria: marcar dezenas de itens como
	// "recomendado" para inflar a sensação de problema.
	eng := novoMotor(t, winreg.NewFake(), false)
	rec := eng.Recommended(tweak.ProfilePersonal)
	total := len(eng.Registry.ForProfile(tweak.ProfilePersonal))
	if len(rec) == 0 {
		t.Fatal("nenhum item recomendado — o app não ajudaria ninguém")
	}
	if len(rec)*2 > total {
		t.Fatalf("%d de %d itens marcados como recomendados: alto demais para um catálogo conservador", len(rec), total)
	}
}

func TestPontoDeRestauracaoQueFalhaAbortaOLote(t *testing.T) {
	f := winreg.NewFake()
	eng := novoMotor(t, f, true)
	eng.CreateRestorePoint = func(string) (uint64, error) {
		return 0, context.DeadlineExceeded // simula Proteção do Sistema desligada
	}

	res := eng.Apply(context.Background(), []string{idUsuario}, false)
	if res[0].Err == nil {
		t.Fatal("sem ponto de restauração, o lote tinha que parar")
	}
	if len(f.Keys()) != 0 {
		t.Fatalf("alterou o sistema mesmo sem conseguir criar a rede de segurança: %v", f.Keys())
	}
}

func TestRevertBatchPreservesManualTweaks(t *testing.T) {
	f := winreg.NewFake()
	eng := novoMotor(t, f, false)
	ctx := context.Background()

	idManual := "visual.menu-show-delay"
	idPerfil := "jogos.gamedvr-usuario"

	// 1. Aplica ajuste manual
	eng.ApplyBatch(ctx, []string{idManual}, false, "manual", "batch-manual-1")

	// 2. Aplica perfil
	batchPerfil := "batch-jogo-1"
	eng.ApplyBatch(ctx, []string{idPerfil}, false, "perfil-jogo", batchPerfil)

	// 3. Reverte apenas o lote do perfil
	res := eng.RevertBatch(ctx, batchPerfil, false)
	if len(res) == 0 || !res[0].Applied {
		t.Fatalf("RevertBatch falhou: %+v", res)
	}

	// 4. Verifica que o tweak manual ainda está pendente/ativo
	pending, err := eng.History.Pending()
	if err != nil {
		t.Fatalf("Pending: %v", err)
	}
	if _, ok := pending[idManual]; !ok {
		t.Error("ajuste manual foi indevidamente removido ao reverter o lote do perfil")
	}
	if _, ok := pending[idPerfil]; ok {
		t.Error("tweak do perfil ainda consta como pendente após RevertBatch")
	}
}
