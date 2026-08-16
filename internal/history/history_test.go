package history_test

import (
	"path/filepath"
	"testing"

	"optimizer/internal/history"
	"optimizer/internal/tweak"
)

func novoStore(t *testing.T) *history.Store {
	t.Helper()
	s, err := history.Open(filepath.Join(t.TempDir(), "history.jsonl"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	return s
}

func TestHistoricoVazioNaoEErro(t *testing.T) {
	s := novoStore(t)
	entradas, err := s.All()
	if err != nil {
		t.Fatalf("All: %v", err)
	}
	if len(entradas) != 0 {
		t.Fatalf("histórico novo veio com %d entradas", len(entradas))
	}
}

func TestPendenteSomeDepoisDeDesfazer(t *testing.T) {
	s := novoStore(t)
	if _, err := s.Append(history.Entry{TweakID: "a", Action: history.ActionApply, Success: true,
		Snapshot: tweak.Snapshot{"x": 1}}); err != nil {
		t.Fatalf("Append: %v", err)
	}

	if _, ok, _ := s.PendingFor("a"); !ok {
		t.Fatal("apply bem-sucedido deveria ficar pendente de desfazer")
	}

	if _, err := s.Append(history.Entry{TweakID: "a", Action: history.ActionRevert, Success: true}); err != nil {
		t.Fatalf("Append: %v", err)
	}
	if _, ok, _ := s.PendingFor("a"); ok {
		t.Fatal("depois de desfazer, o item não pode continuar pendente")
	}
}

func TestSimulacaoENaoIntroduzDesfazer(t *testing.T) {
	s := novoStore(t)
	if _, err := s.Append(history.Entry{TweakID: "a", Action: history.ActionApply, Success: true, DryRun: true}); err != nil {
		t.Fatalf("Append: %v", err)
	}
	if _, ok, _ := s.PendingFor("a"); ok {
		t.Fatal("simulação não alterou nada, logo não há o que desfazer")
	}
	// ...mas continua registrada, para o usuário ver o que o app pensou em fazer.
	entradas, _ := s.All()
	if len(entradas) != 1 {
		t.Fatalf("a simulação não foi registrada no histórico")
	}
}

func TestFalhaNaoContaComoPendente(t *testing.T) {
	s := novoStore(t)
	if _, err := s.Append(history.Entry{TweakID: "a", Action: history.ActionApply, Success: false,
		Error: "acesso negado"}); err != nil {
		t.Fatalf("Append: %v", err)
	}
	if _, ok, _ := s.PendingFor("a"); ok {
		t.Fatal("apply que falhou não pode virar item de desfazer")
	}
}

func TestSnapshotVoltaLegivelDoDisco(t *testing.T) {
	s := novoStore(t)
	snap := tweak.Snapshot{"HKCU\\Caminho!Valor": map[string]any{"exists": true, "dword": 1}}
	if _, err := s.Append(history.Entry{TweakID: "a", Action: history.ActionApply, Success: true, Snapshot: snap}); err != nil {
		t.Fatalf("Append: %v", err)
	}

	e, ok, err := s.PendingFor("a")
	if err != nil || !ok {
		t.Fatalf("PendingFor: ok=%v err=%v", ok, err)
	}
	if _, existe := e.Snapshot["HKCU\\Caminho!Valor"]; !existe {
		t.Fatalf("snapshot voltou incompleto: %#v", e.Snapshot)
	}
}
