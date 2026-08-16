// Package regtweak implementa, uma única vez, o padrão "otimização que é um
// conjunto de valores de registro" — que cobre a maior parte do catálogo.
//
// Um Tweak aqui pode tocar N valores (ex.: desligar aceleração do mouse são 3
// valores independentes). Daí o estado StatePartial existir de verdade: parte
// dos valores no alvo, parte não.
package regtweak

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"optimizer/internal/tweak"
	"optimizer/internal/winreg"
)

type Kind int

const (
	KindDWord Kind = iota
	KindString
)

// Value descreve um valor de registro que o tweak controla.
type Value struct {
	Root winreg.Root
	Path string
	Name string
	Kind Kind

	optimized  any // uint32 (REG_DWORD) ou string (REG_SZ)
	winDefault any // o que o Windows faz quando o valor NÃO existe

	// AlreadyOptimized, se definido, diz se o valor atual já é bom o bastante.
	// Existe para o produto nunca "otimizar" para um valor pior do que o
	// usuário já tinha (ex.: MenuShowDelay já em 0, nosso alvo é 10).
	AlreadyOptimized func(current any) bool
}

// DWord descreve um REG_DWORD. windowsDefault é o comportamento efetivo quando
// o valor não existe no registro — não um chute: precisa vir do catálogo técnico.
func DWord(root winreg.Root, path, name string, optimized, windowsDefault uint32) Value {
	return Value{Root: root, Path: path, Name: name, Kind: KindDWord,
		optimized: optimized, winDefault: windowsDefault}
}

// String descreve um REG_SZ (vários ajustes clássicos do Control Panel são
// string, não DWORD — errar isso é fonte comum de tweak que "não pega").
func String(root winreg.Root, path, name, optimized, windowsDefault string) Value {
	return Value{Root: root, Path: path, Name: name, Kind: KindString,
		optimized: optimized, winDefault: windowsDefault}
}

func (v Value) key() string { return winreg.Key(v.Root, v.Path, v.Name) }

// Human descreve o valor no formato usado nas explicações da UI.
func (v Value) Human() string { return v.key() }

// Spec são os metadados que o próprio código carrega (fallback do manifesto).
// Os nomes de campo diferem dos métodos da interface porque Go não permite
// campo e método com o mesmo nome no mesmo tipo.
type Spec struct {
	TweakID      string
	DisplayName  string
	Explanation  string
	Cat          tweak.Category
	RiskLevel    tweak.Risk
	NeedsRestart bool
	// AppliedText/NotAppliedText são as frases mostradas ao usuário no Check.
	AppliedText    string
	NotAppliedText string
}

func (s Spec) ID() string               { return s.TweakID }
func (s Spec) Name() string             { return s.DisplayName }
func (s Spec) Description() string      { return s.Explanation }
func (s Spec) Category() tweak.Category { return s.Cat }
func (s Spec) Risk() tweak.Risk         { return s.RiskLevel }
func (s Spec) RequiresRestart() bool    { return s.NeedsRestart }

// Tweak é uma otimização composta por um ou mais valores de registro.
type Tweak struct {
	Spec
	Backend winreg.Backend
	Values  []Value
}

var _ tweak.Tweak = Tweak{}

// snapshot do estado anterior de um valor. Exists=false significa "o valor não
// existia" — desfazer, nesse caso, é APAGAR, não gravar o default.
type valueSnapshot struct {
	Exists bool   `json:"exists"`
	DWord  uint32 `json:"dword,omitempty"`
	Str    string `json:"str,omitempty"`
}

func decodeSnapshot(raw any) (valueSnapshot, error) {
	if vs, ok := raw.(valueSnapshot); ok {
		return vs, nil
	}
	// Veio do histórico em JSONL: chegou como map[string]any.
	b, err := json.Marshal(raw)
	if err != nil {
		return valueSnapshot{}, fmt.Errorf("snapshot ilegível: %w", err)
	}
	var vs valueSnapshot
	if err := json.Unmarshal(b, &vs); err != nil {
		return valueSnapshot{}, fmt.Errorf("snapshot ilegível: %w", err)
	}
	return vs, nil
}

// read devolve o valor efetivo atual (o default do Windows quando ausente) e o
// snapshot do que estava lá.
func (t Tweak) read(v Value) (current any, snap valueSnapshot, err error) {
	switch v.Kind {
	case KindDWord:
		n, err := t.Backend.GetDWord(v.Root, v.Path, v.Name)
		if errors.Is(err, winreg.ErrNotExist) {
			return v.winDefault, valueSnapshot{Exists: false}, nil
		}
		if err != nil {
			return nil, valueSnapshot{}, fmt.Errorf("lendo %s: %w", v.key(), err)
		}
		return n, valueSnapshot{Exists: true, DWord: n}, nil
	case KindString:
		s, err := t.Backend.GetString(v.Root, v.Path, v.Name)
		if errors.Is(err, winreg.ErrNotExist) {
			return v.winDefault, valueSnapshot{Exists: false}, nil
		}
		if err != nil {
			return nil, valueSnapshot{}, fmt.Errorf("lendo %s: %w", v.key(), err)
		}
		return s, valueSnapshot{Exists: true, Str: s}, nil
	}
	return nil, valueSnapshot{}, fmt.Errorf("tipo de valor desconhecido em %s", v.key())
}

func (t Tweak) write(v Value) error {
	switch v.Kind {
	case KindDWord:
		return t.Backend.SetDWord(v.Root, v.Path, v.Name, v.optimized.(uint32))
	case KindString:
		return t.Backend.SetString(v.Root, v.Path, v.Name, v.optimized.(string))
	}
	return fmt.Errorf("tipo de valor desconhecido em %s", v.key())
}

func satisfied(v Value, current any) bool {
	if current == v.optimized {
		return true
	}
	if v.AlreadyOptimized != nil {
		return v.AlreadyOptimized(current)
	}
	return false
}

func (t Tweak) Check(ctx context.Context) (tweak.CheckResult, error) {
	if err := ctx.Err(); err != nil {
		return tweak.CheckResult{}, err
	}
	snap := tweak.Snapshot{}
	ok := 0
	var pending []string
	for _, v := range t.Values {
		current, vs, err := t.read(v)
		if err != nil {
			return tweak.CheckResult{}, err
		}
		snap[v.key()] = vs
		if satisfied(v, current) {
			ok++
		} else {
			// O valor medido entra no texto: o produto mostra números reais,
			// nunca "o Windows está no padrão" sem ter conferido.
			pending = append(pending, fmt.Sprintf("%s = %v (alvo %v)", v.Name, current, v.optimized))
		}
	}

	medido := ""
	if len(pending) > 0 {
		medido = " Medimos agora: " + strings.Join(pending, "; ") + "."
	}

	switch {
	case ok == len(t.Values):
		return tweak.CheckResult{State: tweak.StateApplied, Detail: t.appliedText(), RawSnapshot: snap}, nil
	case ok > 0:
		return tweak.CheckResult{
			State:       tweak.StatePartial,
			Detail:      fmt.Sprintf("Aplicado só em parte (%d de %d ajustes).%s", ok, len(t.Values), medido),
			RawSnapshot: snap,
		}, nil
	default:
		return tweak.CheckResult{State: tweak.StateNotApplied, Detail: t.notAppliedText() + medido, RawSnapshot: snap}, nil
	}
}

func (t Tweak) appliedText() string {
	if t.AppliedText != "" {
		return t.AppliedText
	}
	return "Já está aplicado."
}

func (t Tweak) notAppliedText() string {
	if t.NotAppliedText != "" {
		return t.NotAppliedText
	}
	return "Ainda não está aplicado."
}

// Apply é idempotente: reescrever o mesmo valor não muda nada. Em falha parcial
// o snapshot ainda é devolvido junto do erro, para que o histórico consiga
// desfazer o que chegou a ser gravado.
func (t Tweak) Apply(ctx context.Context, dryRun bool) (tweak.ApplyResult, error) {
	current, err := t.Check(ctx)
	if err != nil {
		return tweak.ApplyResult{}, err
	}

	if dryRun {
		return tweak.ApplyResult{
			Snapshot:      current.RawSnapshot,
			RestartNeeded: t.NeedsRestart,
			Detail:        "Simulação — nada foi gravado. " + t.plan(current),
		}, nil
	}

	var errs []error
	for _, v := range t.Values {
		if err := t.write(v); err != nil {
			errs = append(errs, fmt.Errorf("gravando %s: %w", v.key(), err))
		}
	}
	res := tweak.ApplyResult{
		Snapshot:      current.RawSnapshot,
		RestartNeeded: t.NeedsRestart,
		Detail:        t.appliedText(),
	}
	if len(errs) > 0 {
		res.Detail = "Não foi possível aplicar tudo."
		return res, errors.Join(errs...)
	}
	return res, nil
}

func (t Tweak) plan(current tweak.CheckResult) string {
	if current.State == tweak.StateApplied {
		return "Nada mudaria: já está no valor otimizado."
	}
	parts := make([]string, 0, len(t.Values))
	for _, v := range t.Values {
		parts = append(parts, fmt.Sprintf("%s = %v", v.key(), v.optimized))
	}
	return "Seria gravado: " + strings.Join(parts, "; ") + "."
}

// Revert usa o snapshot gravado no histórico. Valor que não existia antes é
// apagado (voltar ao default de fábrica é diferente de gravar o valor default).
func (t Tweak) Revert(ctx context.Context, snap tweak.Snapshot, dryRun bool) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	var errs []error
	for _, v := range t.Values {
		raw, ok := snap[v.key()]
		if !ok {
			errs = append(errs, fmt.Errorf("snapshot não tem o estado anterior de %s", v.key()))
			continue
		}
		vs, err := decodeSnapshot(raw)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", v.key(), err))
			continue
		}
		if dryRun {
			continue
		}
		errs = append(errs, t.restore(v, vs))
	}
	return errors.Join(errs...)
}

func (t Tweak) restore(v Value, vs valueSnapshot) error {
	if !vs.Exists {
		err := t.Backend.DeleteValue(v.Root, v.Path, v.Name)
		if err == nil || errors.Is(err, winreg.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("apagando %s: %w", v.key(), err)
	}
	var err error
	switch v.Kind {
	case KindDWord:
		err = t.Backend.SetDWord(v.Root, v.Path, v.Name, vs.DWord)
	case KindString:
		err = t.Backend.SetString(v.Root, v.Path, v.Name, vs.Str)
	}
	if err != nil {
		return fmt.Errorf("restaurando %s: %w", v.key(), err)
	}
	return nil
}

func (t Tweak) Verify(ctx context.Context) (tweak.VerifyResult, error) {
	res, err := t.Check(ctx)
	if err != nil {
		return tweak.VerifyResult{}, err
	}
	return tweak.VerifyResult{Success: res.State == tweak.StateApplied, Detail: res.Detail}, nil
}
