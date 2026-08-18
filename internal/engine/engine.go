// Package engine orquestra o catálogo: diagnosticar, aplicar em lote, verificar
// e desfazer — sempre gravando no histórico antes de considerar a tarefa pronta.
package engine

import (
	"context"
	"errors"
	"fmt"

	"optimizer/internal/history"
	"optimizer/internal/tweak"
)

// Engine amarra catálogo + histórico. As dependências de sistema (elevação,
// ponto de restauração) entram por injeção para os testes não tocarem o Windows.
type Engine struct {
	Registry *tweak.Registry
	History  *history.Store

	// Elevated diz se o processo atual já é administrador.
	Elevated bool
	// CreateRestorePoint, quando definido, é chamado UMA vez por lote real
	// (nunca em simulação) antes da primeira alteração.
	CreateRestorePoint func(description string) (uint64, error)
}

// Status é a linha do diagnóstico mostrada na UI.
type Status struct {
	Meta   tweak.Meta
	State  tweak.State
	Detail string
	// CanUndo indica que existe um Apply no histórico ainda não desfeito.
	CanUndo bool
	Err     error
}

// Result é o desfecho de um Apply/Revert de um item.
type Result struct {
	ID            string
	Name          string
	Action        history.Action
	Applied       bool
	Skipped       bool
	Reason        string
	Detail        string
	RestartNeeded bool
	Err           error
}

// Scan diagnostica todos os itens do perfil. Nunca altera nada.
func (e *Engine) Scan(ctx context.Context, profile tweak.Profile) []Status {
	pending, err := e.History.Pending()
	if err != nil {
		pending = map[string]history.Entry{}
	}

	entries := e.Registry.ForProfile(profile)
	out := make([]Status, 0, len(entries))
	for _, entry := range entries {
		s := Status{Meta: entry.Meta}
		if _, ok := pending[entry.Meta.ID]; ok {
			s.CanUndo = true
		}
		res, err := entry.Tweak.Check(ctx)
		if err != nil {
			s.State = tweak.StateUnknown
			s.Err = err
			s.Detail = friendlyCheckError(entry.Meta, err)
		} else {
			s.State = res.State
			s.Detail = res.Detail
		}
		out = append(out, s)
	}
	return out
}

func friendlyCheckError(m tweak.Meta, err error) string {
	if m.RequiresAdmin {
		return "Não deu para ler o estado atual — este item mexe em configuração do computador inteiro e a leitura pode exigir administrador."
	}
	return fmt.Sprintf("Não deu para ler o estado atual: %v", err)
}

// Recommended devolve os IDs marcados como recomendados no perfil.
func (e *Engine) Recommended(profile tweak.Profile) []string {
	var ids []string
	for _, entry := range e.Registry.ForProfile(profile) {
		if entry.Meta.RecommendedDefault {
			ids = append(ids, entry.Meta.ID)
		}
	}
	return ids
}

// Apply aplica os IDs pedidos, em lote. Com dryRun a máquina não é tocada.
//
// Ordem por item: checar → pular se já aplicado → aplicar → verificar → gravar
// no histórico. O histórico é gravado inclusive em falha, porque uma falha
// parcial ainda pode ter deixado valores gravados que precisam ser desfeitos.
func (e *Engine) Apply(ctx context.Context, ids []string, dryRun bool) []Result {
	results := make([]Result, 0, len(ids))
	restorePointDone := false

	for _, id := range ids {
		t, ok := e.Registry.Known(id)
		if !ok {
			results = append(results, Result{ID: id, Action: history.ActionApply,
				Err: fmt.Errorf("otimização desconhecida: %s", id)})
			continue
		}
		meta, _ := e.Registry.Meta(id)
		res := Result{ID: id, Name: meta.DisplayName, Action: history.ActionApply}

		if meta.RequiresAdmin && !e.Elevated {
			res.Skipped = true
			res.Reason = "precisa de permissão de administrador"
			results = append(results, res)
			continue
		}

		current, err := t.Check(ctx)
		if err != nil {
			res.Err = err
			results = append(results, res)
			continue
		}
		if current.State == tweak.StateApplied {
			res.Skipped = true
			res.Reason = "já estava aplicado"
			res.Detail = current.Detail
			results = append(results, res)
			continue
		}

		if !dryRun && !restorePointDone && e.CreateRestorePoint != nil {
			// Um ponto de restauração por lote, nunca um por item.
			if _, err := e.CreateRestorePoint("Optimizer — antes de aplicar otimizações"); err != nil {
				res.Err = fmt.Errorf("não foi possível criar o ponto de restauração, nada foi alterado: %w", err)
				results = append(results, res)
				return results
			}
			restorePointDone = true
		}

		applyRes, applyErr := t.Apply(ctx, dryRun)
		res.Detail = applyRes.Detail
		res.RestartNeeded = applyRes.RestartNeeded
		res.Err = applyErr
		res.Applied = applyErr == nil

		if applyErr == nil && !dryRun {
			v, err := t.Verify(ctx)
			switch {
			case err != nil:
				res.Err = fmt.Errorf("aplicado, mas não deu para confirmar: %w", err)
				res.Applied = false
			case !v.Success:
				res.Err = errors.New("a alteração foi gravada mas o Windows não confirmou o novo valor")
				res.Applied = false
			default:
				res.Detail = v.Detail
			}
		}

		entry := history.Entry{
			TweakID:       id,
			Action:        history.ActionApply,
			DryRun:        dryRun,
			Success:       res.Err == nil,
			Detail:        res.Detail,
			Snapshot:      applyRes.Snapshot,
			RestartNeeded: applyRes.RestartNeeded,
		}
		if res.Err != nil {
			entry.Error = res.Err.Error()
		}
		if _, err := e.History.Append(entry); err != nil && res.Err == nil {
			res.Err = fmt.Errorf("alteração feita, mas o histórico não foi gravado: %w", err)
		}

		results = append(results, res)
	}
	return results
}

// Revert desfaz, usando o snapshot gravado no histórico. Sem snapshot não há
// desfazer: o app não "adivinha" o valor de fábrica.
func (e *Engine) Revert(ctx context.Context, ids []string, dryRun bool) []Result {
	results := make([]Result, 0, len(ids))
	for _, id := range ids {
		t, ok := e.Registry.Known(id)
		if !ok {
			results = append(results, Result{ID: id, Action: history.ActionRevert,
				Err: fmt.Errorf("otimização desconhecida: %s", id)})
			continue
		}
		meta, _ := e.Registry.Meta(id)
		res := Result{ID: id, Name: meta.DisplayName, Action: history.ActionRevert}

		prev, ok, err := e.History.PendingFor(id)
		if err != nil {
			res.Err = err
			results = append(results, res)
			continue
		}
		if !ok {
			res.Skipped = true
			res.Reason = "não há alteração deste item para desfazer"
			results = append(results, res)
			continue
		}
		if meta.RequiresAdmin && !e.Elevated {
			res.Skipped = true
			res.Reason = "precisa de permissão de administrador"
			results = append(results, res)
			continue
		}

		if err := t.Revert(ctx, prev.Snapshot, dryRun); err != nil {
			res.Err = err
		} else {
			res.Applied = true
			res.Detail = "Voltou ao estado anterior."
			res.RestartNeeded = prev.RestartNeeded
		}

		entry := history.Entry{
			TweakID: id,
			Action:  history.ActionRevert,
			DryRun:  dryRun,
			Success: res.Err == nil,
			Detail:  res.Detail,
		}
		if res.Err != nil {
			entry.Error = res.Err.Error()
		}
		if _, err := e.History.Append(entry); err != nil && res.Err == nil {
			res.Err = fmt.Errorf("desfeito, mas o histórico não foi gravado: %w", err)
		}
		results = append(results, res)
	}
	return results
}

// PendingIDs lista o que pode ser desfeito agora (para o botão "desfazer tudo").
func (e *Engine) PendingIDs() ([]string, error) {
	pending, err := e.History.Pending()
	if err != nil {
		return nil, err
	}
	var ids []string
	for _, entry := range e.Registry.Catalog() {
		if _, ok := pending[entry.Meta.ID]; ok {
			ids = append(ids, entry.Meta.ID)
		}
	}
	return ids, nil
}

// ApplyProfile aplica um perfil nomeado (ex: "Rede Rápida"). É um wrapper que
// resolve os IDs do perfil e chama Apply() em lote.
func (e *Engine) ApplyProfile(ctx context.Context, profile NetworkProfile, dryRun bool) []Result {
	return e.Apply(ctx, profile.TweakIDs, dryRun)
}

// NetworkProfile é uma cópia local de profiles.NetworkProfile para evitar import cíclico.
// Ver internal/profiles/netprofiles.go.
type NetworkProfile struct {
	Key      string
	Name     string
	TweakIDs []string
	Caveats  string
}
