package regtweak_test

import (
	"context"
	"encoding/json"
	"testing"

	"optimizer/internal/tweak"
	"optimizer/internal/tweaks/regtweak"
	"optimizer/internal/winreg"
)

const (
	testPath = `Software\OptimizerTests\Exemplo`
	dwordVal = "Ligado"
	szVal    = "Atraso"
)

func novoDWord(b winreg.Backend) regtweak.Tweak {
	return regtweak.Tweak{
		Spec: regtweak.Spec{
			TweakID:     "teste.dword",
			DisplayName: "Exemplo DWORD",
			Cat:         tweak.CategoryVisualEffects,
			RiskLevel:   tweak.RiskLow,
		},
		Backend: b,
		Values:  []regtweak.Value{regtweak.DWord(winreg.HKCU, testPath, dwordVal, 0, 1)},
	}
}

func TestValorAusenteContaComoPadraoDoWindows(t *testing.T) {
	f := winreg.NewFake()
	tw := novoDWord(f)

	res, err := tw.Check(context.Background())
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	// Valor ausente = Windows no padrão (1) = tweak não aplicado.
	if res.State != tweak.StateNotApplied {
		t.Fatalf("estado = %v, queria StateNotApplied", res.State)
	}
}

func TestDesfazerApagaValorQueNaoExistiaAntes(t *testing.T) {
	f := winreg.NewFake()
	tw := novoDWord(f)
	ctx := context.Background()

	res, err := tw.Apply(ctx, false)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if got, err := f.GetDWord(winreg.HKCU, testPath, dwordVal); err != nil || got != 0 {
		t.Fatalf("depois do Apply: valor=%v err=%v, queria 0", got, err)
	}

	if err := tw.Revert(ctx, res.Snapshot, false); err != nil {
		t.Fatalf("Revert: %v", err)
	}
	// O valor não existia antes: desfazer precisa APAGAR, não gravar o default.
	if _, err := f.GetDWord(winreg.HKCU, testPath, dwordVal); err == nil {
		t.Fatal("o valor continuou existindo depois de desfazer")
	}
}

func TestDesfazerRestauraValorAnterior(t *testing.T) {
	f := winreg.NewFake()
	f.Seed(winreg.HKCU, testPath, dwordVal, uint32(1))
	tw := novoDWord(f)
	ctx := context.Background()

	res, err := tw.Apply(ctx, false)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if err := tw.Revert(ctx, res.Snapshot, false); err != nil {
		t.Fatalf("Revert: %v", err)
	}
	if got, _ := f.GetDWord(winreg.HKCU, testPath, dwordVal); got != 1 {
		t.Fatalf("valor restaurado = %d, queria 1", got)
	}
}

func TestSnapshotSobreviveAoJSONDoHistorico(t *testing.T) {
	f := winreg.NewFake()
	f.Seed(winreg.HKCU, testPath, dwordVal, uint32(1))
	tw := novoDWord(f)
	ctx := context.Background()

	res, err := tw.Apply(ctx, false)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}

	// O histórico é JSONL: o snapshot volta como map[string]any, não como o
	// struct original. Desfazer precisa continuar funcionando depois disso.
	raw, err := json.Marshal(res.Snapshot)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var voltou tweak.Snapshot
	if err := json.Unmarshal(raw, &voltou); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if err := tw.Revert(ctx, voltou, false); err != nil {
		t.Fatalf("Revert com snapshot vindo do JSON: %v", err)
	}
	if got, _ := f.GetDWord(winreg.HKCU, testPath, dwordVal); got != 1 {
		t.Fatalf("valor restaurado = %d, queria 1", got)
	}
}

func TestSimulacaoNaoGravaNada(t *testing.T) {
	f := winreg.NewFake()
	tw := novoDWord(f)

	if _, err := tw.Apply(context.Background(), true); err != nil {
		t.Fatalf("Apply dry-run: %v", err)
	}
	if len(f.Keys()) != 0 {
		t.Fatalf("a simulação gravou %v", f.Keys())
	}
}

func TestEstadoParcialComVariosValores(t *testing.T) {
	f := winreg.NewFake()
	tw := regtweak.Tweak{
		Spec:    regtweak.Spec{TweakID: "teste.multi", DisplayName: "Multi"},
		Backend: f,
		Values: []regtweak.Value{
			regtweak.String(winreg.HKCU, testPath, "A", "0", "1"),
			regtweak.String(winreg.HKCU, testPath, "B", "0", "6"),
		},
	}
	f.Seed(winreg.HKCU, testPath, "A", "0") // só um dos dois no alvo

	res, err := tw.Check(context.Background())
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if res.State != tweak.StatePartial {
		t.Fatalf("estado = %v, queria StatePartial", res.State)
	}
}

func TestNaoPioraValorQueJaEstaMelhorQueOAlvo(t *testing.T) {
	f := winreg.NewFake()
	v := regtweak.String(winreg.HKCU, testPath, szVal, "10", "400")
	v.AlreadyOptimized = func(cur any) bool { return cur == "0" }
	tw := regtweak.Tweak{
		Spec:    regtweak.Spec{TweakID: "teste.sz", DisplayName: "SZ"},
		Backend: f,
		Values:  []regtweak.Value{v},
	}
	f.Seed(winreg.HKCU, testPath, szVal, "0")

	res, err := tw.Check(context.Background())
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if res.State != tweak.StateApplied {
		t.Fatalf("estado = %v, queria StateApplied (0 já é melhor que 10)", res.State)
	}
}

func TestAcessoNegadoDevolveErroSemGravar(t *testing.T) {
	f := winreg.NewFake()
	f.DenyPrefix("HKLM\\")
	tw := regtweak.Tweak{
		Spec:    regtweak.Spec{TweakID: "teste.hklm", DisplayName: "HKLM"},
		Backend: f,
		Values:  []regtweak.Value{regtweak.DWord(winreg.HKLM, testPath, dwordVal, 0, 1)},
	}

	if _, err := tw.Check(context.Background()); err == nil {
		t.Fatal("Check deveria falhar com acesso negado")
	}
}
