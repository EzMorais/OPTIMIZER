package netdiag_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"optimizer/internal/netdiag"
	"optimizer/internal/tweak"
)

type fakeMTU struct {
	valor    uint32
	semAdmin bool
}

func (f *fakeMTU) Get(uint64) (uint32, error) { return f.valor, nil }
func (f *fakeMTU) Set(_ uint64, v uint32) error {
	if f.semAdmin {
		return errors.New("acesso negado")
	}
	f.valor = v
	return nil
}

func novoAjuste(ctl netdiag.MTUController) (netdiag.MTUTweak, error) {
	iface := netdiag.Interface{Name: "Ethernet", Luid: 0xABCD, MTU: 1500}
	rep := netdiag.Report{Host: "8.8.8.8", PathMTU: 1480, LargestOK: 1452}
	return netdiag.NewMTUTweak(iface, rep, netdiag.Diagnose(rep, &iface), ctl)
}

func TestAjusteDeMTUSoExisteDepoisDaMedicao(t *testing.T) {
	iface := netdiag.Interface{Name: "Ethernet", MTU: 1500}
	rep := netdiag.Report{Host: "8.8.8.8", PathMTU: 1500, LargestOK: 1472}
	_, err := netdiag.NewMTUTweak(iface, rep, netdiag.Diagnose(rep, &iface), &fakeMTU{valor: 1500})
	if err == nil {
		t.Fatal("sem diagnóstico que peça mudança, não pode existir ajuste de MTU para aplicar")
	}
}

func TestAplicarEDesfazerMTU(t *testing.T) {
	ctl := &fakeMTU{valor: 1500}
	tw, err := novoAjuste(ctl)
	if err != nil {
		t.Fatalf("montando ajuste: %v", err)
	}
	ctx := context.Background()

	res, err := tw.Apply(ctx, false)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if ctl.valor != 1480 {
		t.Fatalf("MTU depois do apply = %d, queria 1480", ctl.valor)
	}
	if v, _ := tw.Verify(ctx); !v.Success {
		t.Fatal("Verify não confirmou a alteração")
	}

	// O snapshot passa pelo histórico em JSON antes de voltar para o Revert.
	raw, _ := json.Marshal(res.Snapshot)
	var voltou tweak.Snapshot
	if err := json.Unmarshal(raw, &voltou); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if err := tw.Revert(ctx, voltou, false); err != nil {
		t.Fatalf("Revert: %v", err)
	}
	if ctl.valor != 1500 {
		t.Fatalf("MTU depois do revert = %d, queria 1500", ctl.valor)
	}
	if _, ok := res.Snapshot["mtu.destino"]; !ok {
		t.Fatal("snapshot não preservou o MTU de destino para reidratação após reinício")
	}
}

func TestSimulacaoDeMTUNaoAlteraNada(t *testing.T) {
	ctl := &fakeMTU{valor: 1500}
	tw, err := novoAjuste(ctl)
	if err != nil {
		t.Fatalf("montando ajuste: %v", err)
	}
	if _, err := tw.Apply(context.Background(), true); err != nil {
		t.Fatalf("Apply dry-run: %v", err)
	}
	if ctl.valor != 1500 {
		t.Fatalf("a simulação alterou o MTU para %d", ctl.valor)
	}
}

func TestIDdoAjusteIdentificaOAdaptador(t *testing.T) {
	tw, err := novoAjuste(&fakeMTU{valor: 1500})
	if err != nil {
		t.Fatalf("montando ajuste: %v", err)
	}
	// O LUID é estável entre reinícios (o índice da interface não é): é o que
	// garante que "desfazer" vai mexer na placa certa depois de reiniciar.
	if tw.ID() != "rede.mtu.000000000000abcd" {
		t.Fatalf("ID = %q, não identifica o adaptador de forma estável", tw.ID())
	}
}
