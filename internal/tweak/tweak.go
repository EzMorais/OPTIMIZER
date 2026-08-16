// Package tweak define o contrato que toda otimização do catálogo implementa.
// Ver docs/arquitetura-app-desktop.md, seção 4.
package tweak

import (
	"context"
	"sort"
)

// State representa o estado atual de uma otimização no computador do usuário.
type State int

const (
	StateUnknown State = iota
	StateApplied
	StateNotApplied
	StatePartial
)

// Risk classifica o quão seguro/reversível é um Tweak. O catálogo padrão
// (conservador) só deve conter RiskLow e RiskMedium.
type Risk int

const (
	RiskLow Risk = iota
	RiskMedium
	RiskHigh
)

type Category string

const (
	CategoryStartup            Category = "startup"
	CategoryVisualEffects      Category = "visual_effects"
	CategoryPower              Category = "power"
	CategoryPrivacyNonSecurity Category = "privacy_non_security"
	CategoryStorage            Category = "storage"
	CategoryNetwork            Category = "network"
	CategoryGaming             Category = "gaming"
	CategoryInput              Category = "input"
	CategorySystem             Category = "system"
)

// CategoryLabel é o nome mostrado ao usuário (PT-BR, sem jargão).
func CategoryLabel(c Category) string {
	switch c {
	case CategoryStartup:
		return "Inicialização"
	case CategoryVisualEffects:
		return "Efeitos visuais"
	case CategoryPower:
		return "Energia"
	case CategoryPrivacyNonSecurity:
		return "Privacidade"
	case CategoryStorage:
		return "Armazenamento"
	case CategoryNetwork:
		return "Internet e rede"
	case CategoryGaming:
		return "Jogos"
	case CategoryInput:
		return "Mouse e teclado"
	case CategorySystem:
		return "Sistema"
	}
	return string(c)
}

func (s State) String() string {
	switch s {
	case StateApplied:
		return "aplicado"
	case StateNotApplied:
		return "não aplicado"
	case StatePartial:
		return "parcial"
	}
	return "desconhecido"
}

func (r Risk) String() string {
	switch r {
	case RiskLow:
		return "baixo"
	case RiskMedium:
		return "médio"
	case RiskHigh:
		return "alto"
	}
	return "desconhecido"
}

// Snapshot é o estado anterior necessário para desfazer a mudança.
// Precisa ser serializável em JSON, pois entra no histórico em disco.
type Snapshot map[string]any

type CheckResult struct {
	State       State
	Detail      string
	RawSnapshot Snapshot
}

type ApplyResult struct {
	Snapshot      Snapshot
	RestartNeeded bool
	Detail        string
}

type VerifyResult struct {
	Success bool
	Detail  string
}

// Tweak é o contrato que toda otimização do catálogo implementa.
type Tweak interface {
	ID() string
	Name() string
	Description() string
	Category() Category
	Risk() Risk
	RequiresRestart() bool

	// Check inspeciona o computador do usuário SEM alterar nada.
	Check(ctx context.Context) (CheckResult, error)

	// Apply aplica a mudança. Deve ser idempotente. Se dryRun=true,
	// faz todas as validações e retorna o que ACONTECERIA, sem escrever nada.
	Apply(ctx context.Context, dryRun bool) (ApplyResult, error)

	// Revert desfaz usando o snapshot gravado no histórico.
	Revert(ctx context.Context, snapshot Snapshot, dryRun bool) error

	// Verify confirma, depois do Apply, que a mudança realmente colou.
	Verify(ctx context.Context) (VerifyResult, error)
}

// Entry é o par (metadado, implementação) de um item do catálogo.
type Entry struct {
	Meta  Meta
	Tweak Tweak
}

// Registry casa Meta (dado, pode vir de fora) com Tweak (código, sempre local).
type Registry struct {
	impls map[string]Tweak
	metas map[string]Meta
}

func NewRegistry() *Registry {
	return &Registry{impls: map[string]Tweak{}, metas: map[string]Meta{}}
}

// Register adiciona a implementação compilada e o metadado padrão embutido no
// binário. Campos de texto vazios caem para o que o próprio Tweak declara.
func (r *Registry) Register(t Tweak, m Meta) {
	id := t.ID()
	m.ID = id
	if m.DisplayName == "" {
		m.DisplayName = t.Name()
	}
	if m.Description == "" {
		m.Description = t.Description()
	}
	r.impls[id] = t
	r.metas[id] = m
}

// Known ignora silenciosamente qualquer ID que não corresponda a um Tweak
// já compilado — o registro nunca executa o que vier de um manifesto remoto.
func (r *Registry) Known(id string) (Tweak, bool) {
	t, ok := r.impls[id]
	return t, ok
}

func (r *Registry) Meta(id string) (Meta, bool) {
	m, ok := r.metas[id]
	return m, ok
}

// Catalog devolve os itens habilitados, ordenados por SortOrder e depois por ID
// (ordem estável — mapa em Go não tem ordem).
func (r *Registry) Catalog() []Entry {
	out := make([]Entry, 0, len(r.impls))
	for id, t := range r.impls {
		m := r.metas[id]
		if !m.Enabled {
			continue
		}
		out = append(out, Entry{Meta: m, Tweak: t})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Meta.SortOrder != out[j].Meta.SortOrder {
			return out[i].Meta.SortOrder < out[j].Meta.SortOrder
		}
		return out[i].Meta.ID < out[j].Meta.ID
	})
	return out
}

// ForProfile filtra o catálogo pela aba (Pessoal/Trabalho).
func (r *Registry) ForProfile(p Profile) []Entry {
	all := r.Catalog()
	out := make([]Entry, 0, len(all))
	for _, e := range all {
		if e.Meta.Profiles.Has(p) {
			out = append(out, e)
		}
	}
	return out
}

// ApplyManifest sobrepõe metadados vindos de um manifesto remoto assinado.
// IDs desconhecidos são ignorados — adicionar um tipo NOVO de otimização exige
// nova versão do .exe; reordenar, reescrever texto ou desativar um item não.
// Devolve quantos itens foram efetivamente atualizados.
func (r *Registry) ApplyManifest(metas []Meta) int {
	applied := 0
	for _, m := range metas {
		if _, ok := r.impls[m.ID]; !ok {
			continue
		}
		cur := r.metas[m.ID]
		if m.DisplayName == "" {
			m.DisplayName = cur.DisplayName
		}
		if m.Description == "" {
			m.Description = cur.Description
		}
		if m.Evidence == "" {
			m.Evidence = cur.Evidence
		}
		// RequiresAdmin é propriedade do código, não do manifesto: um manifesto
		// hostil não pode fazer o app achar que um tweak de HKLM dispensa UAC.
		m.RequiresAdmin = cur.RequiresAdmin
		r.metas[m.ID] = m
		applied++
	}
	return applied
}

func (r *Registry) All() []Tweak {
	out := make([]Tweak, 0, len(r.impls))
	for _, t := range r.impls {
		out = append(out, t)
	}
	return out
}
