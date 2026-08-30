package netdiag

import (
	"context"
	"encoding/json"
	"fmt"

	"optimizer/internal/tweak"
)

// MTUController é a fronteira mockável com a configuração de MTU do Windows.
type MTUController interface {
	Get(luid uint64) (uint32, error)
	Set(luid uint64, mtu uint32) error
}

// MTUTweak é criado em tempo de execução, DEPOIS da medição — nunca faz parte
// do catálogo fixo. É a diferença entre "ajustamos porque medimos" e o botão
// mágico de MTU que os concorrentes oferecem sem medir nada.
type MTUTweak struct {
	Iface   Interface
	Target  uint32
	Ctl     MTUController
	Host    string // destino usado na medição, para a explicação
	PathMTU int    // MTU medido do caminho
}

var _ tweak.Tweak = MTUTweak{}

const mtuSnapshotKey = "mtu.anterior"
const mtuTargetKey = "mtu.destino"

// NewMTUTweak monta o ajuste a partir de um diagnóstico. Só faz sentido quando
// o veredito pede mudança.
func NewMTUTweak(iface Interface, rep Report, diag Diagnosis, ctl MTUController) (MTUTweak, error) {
	if diag.SuggestedMTU == 0 {
		return MTUTweak{}, fmt.Errorf("a medição não indicou mudança de MTU — nada a aplicar")
	}
	return MTUTweak{Iface: iface, Target: diag.SuggestedMTU, Ctl: ctl, Host: rep.Host, PathMTU: rep.PathMTU}, nil
}

// ID inclui o LUID do adaptador: é estável entre reinícios (diferente do índice)
// e permite desfazer o ajuste da placa certa mesmo com várias placas na máquina.
func (t MTUTweak) ID() string { return fmt.Sprintf("rede.mtu.%016x", t.Iface.Luid) }

func (t MTUTweak) Name() string {
	return fmt.Sprintf("Ajustar o MTU de %q para %d", t.Iface.Name, t.Target)
}

func (t MTUTweak) Description() string {
	return fmt.Sprintf(
		"Medimos que o caminho até %s só entrega pacotes de até %d bytes sem quebrar, "+
			"enquanto o adaptador %q está configurado em %d. Este ajuste alinha os dois valores.",
		t.Host, t.PathMTU, t.Iface.Name, t.Iface.MTU)
}

func (t MTUTweak) Category() tweak.Category { return tweak.CategoryNetwork }
func (t MTUTweak) Risk() tweak.Risk         { return tweak.RiskMedium }
func (t MTUTweak) RequiresRestart() bool    { return false }

func (t MTUTweak) Check(ctx context.Context) (tweak.CheckResult, error) {
	if err := ctx.Err(); err != nil {
		return tweak.CheckResult{}, err
	}
	current, err := t.Ctl.Get(t.Iface.Luid)
	if err != nil {
		return tweak.CheckResult{}, err
	}
	snap := tweak.Snapshot{mtuSnapshotKey: current, mtuTargetKey: t.Target}
	if current == t.Target {
		return tweak.CheckResult{
			State:       tweak.StateApplied,
			Detail:      fmt.Sprintf("O adaptador %q já está em MTU %d.", t.Iface.Name, current),
			RawSnapshot: snap,
		}, nil
	}
	return tweak.CheckResult{
		State:       tweak.StateNotApplied,
		Detail:      fmt.Sprintf("O adaptador %q está em MTU %d.", t.Iface.Name, current),
		RawSnapshot: snap,
	}, nil
}

func (t MTUTweak) Apply(ctx context.Context, dryRun bool) (tweak.ApplyResult, error) {
	current, err := t.Check(ctx)
	if err != nil {
		return tweak.ApplyResult{}, err
	}
	if dryRun {
		return tweak.ApplyResult{
			Snapshot: current.RawSnapshot,
			Detail: fmt.Sprintf("Simulação — nada foi alterado. O MTU de %q passaria de %v para %d.",
				t.Iface.Name, current.RawSnapshot[mtuSnapshotKey], t.Target),
		}, nil
	}
	if err := t.Ctl.Set(t.Iface.Luid, t.Target); err != nil {
		return tweak.ApplyResult{Snapshot: current.RawSnapshot}, err
	}
	return tweak.ApplyResult{
		Snapshot: current.RawSnapshot,
		Detail:   fmt.Sprintf("MTU de %q ajustado para %d.", t.Iface.Name, t.Target),
	}, nil
}

func (t MTUTweak) Revert(ctx context.Context, snap tweak.Snapshot, dryRun bool) error {
	raw, ok := snap[mtuSnapshotKey]
	if !ok {
		return fmt.Errorf("snapshot sem o MTU anterior de %q", t.Iface.Name)
	}
	prev, err := toUint32(raw)
	if err != nil {
		return fmt.Errorf("snapshot de MTU ilegível: %w", err)
	}
	if dryRun {
		return nil
	}
	return t.Ctl.Set(t.Iface.Luid, prev)
}

func (t MTUTweak) Verify(ctx context.Context) (tweak.VerifyResult, error) {
	res, err := t.Check(ctx)
	if err != nil {
		return tweak.VerifyResult{}, err
	}
	return tweak.VerifyResult{Success: res.State == tweak.StateApplied, Detail: res.Detail}, nil
}

// toUint32 aceita tanto o valor em memória quanto o que voltou do histórico em
// JSON (onde todo número vira float64).
func toUint32(raw any) (uint32, error) {
	switch v := raw.(type) {
	case uint32:
		return v, nil
	case float64:
		return uint32(v), nil
	case json.Number:
		n, err := v.Int64()
		return uint32(n), err
	case int:
		return uint32(v), nil
	}
	return 0, fmt.Errorf("tipo inesperado: %T", raw)
}
