package tweak

import "strings"

// Profile separa as duas abas do produto — o diferencial central contra os
// concorrentes (ver docs/decisoes.md). No perfil Trabalho a barra é mais alta:
// nada que a TI da empresa possa depender entra no padrão.
type Profile uint8

const (
	ProfilePersonal Profile = 1 << iota
	ProfileWork
)

const ProfileBoth = ProfilePersonal | ProfileWork

func (p Profile) Has(other Profile) bool { return p&other != 0 }

func (p Profile) String() string {
	var parts []string
	if p.Has(ProfilePersonal) {
		parts = append(parts, "pessoal")
	}
	if p.Has(ProfileWork) {
		parts = append(parts, "trabalho")
	}
	if len(parts) == 0 {
		return "nenhum"
	}
	return strings.Join(parts, "+")
}

// ParseProfile aceita "pessoal"/"trabalho" (usado pela CLI e, mais adiante, pela UI).
func ParseProfile(s string) (Profile, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "pessoal", "personal":
		return ProfilePersonal, true
	case "trabalho", "work":
		return ProfileWork, true
	case "", "ambos", "todos", "all":
		return ProfileBoth, true
	}
	return 0, false
}

// Meta é o lado "dado" do catálogo: pode ser reescrito por um manifesto remoto
// assinado sem exigir nova versão do .exe. O lado "código" (Tweak) é sempre
// local e compilado — ver docs/arquitetura-app-desktop.md, seção 4.
type Meta struct {
	ID          string `json:"id"`
	Enabled     bool   `json:"enabled"`
	DisplayName string `json:"displayName"`
	Description string `json:"description"`
	// RecommendedDefault marca o que entra no "aplicar recomendados". Só é true
	// para itens com ganho real medido — nunca para encher a lista.
	RecommendedDefault bool `json:"recommendedDefault"`
	// Profiles diz em quais abas o item aparece.
	Profiles Profile `json:"profiles"`
	// RequiresAdmin permite avisar antes do UAC, sem tentar e falhar.
	RequiresAdmin bool `json:"requiresAdmin"`
	// Caveat é o aviso honesto exibido junto do toggle (ganho marginal, efeito
	// variável, ressalva de compatibilidade). Vazio quando não há ressalva.
	Caveat string `json:"caveat,omitempty"`
	// Evidence aponta a seção do catálogo técnico que embasa o item.
	Evidence      string `json:"evidence,omitempty"`
	SortOrder     int    `json:"sortOrder"`
	MinAppVersion string `json:"minAppVersion,omitempty"`
}
