package profiles

import (
	"testing"
)

func TestListarPerfisUso(t *testing.T) {
	perfis := ListarPerfisUso()
	if len(perfis) < 2 {
		t.Fatalf("esperado pelo menos 2 perfis de uso (JOGO e CODING), obteve %d", len(perfis))
	}

	chaves := make(map[string]bool)
	for _, p := range perfis {
		if p.Key == "" || p.Nome == "" || p.Objetivo == "" {
			t.Errorf("perfil com campos obrigatórios vazios: %+v", p)
		}
		if len(p.TweakIDs) == 0 {
			t.Errorf("perfil %s sem tweaks definidos", p.Key)
		}
		if chaves[p.Key] {
			t.Errorf("chave duplicada: %s", p.Key)
		}
		chaves[p.Key] = true
	}
}

func TestCodingProfileNeverDisablesServices(t *testing.T) {
	coding, ok := ObterPerfilUso("coding")
	if !ok {
		t.Fatal("perfil coding não encontrado")
	}

	if len(coding.ServicesToPause) > 0 {
		t.Errorf("CODING nunca deve pausar serviços, mas tem: %v", coding.ServicesToPause)
	}

	for _, s := range coding.ServicesToEnsure {
		if s == "wslservice" || s == "com.docker.service" || s == "vmms" {
			return
		}
	}
	t.Error("CODING deve garantir serviços de WSL/Docker/Hyper-V")
}

func TestJogoProfileHasGameTweaks(t *testing.T) {
	jogo, ok := ObterPerfilUso("jogo")
	if !ok {
		t.Fatal("perfil jogo não encontrado")
	}

	hasGameMode := false
	for _, id := range jogo.TweakIDs {
		if id == "jogos.game-mode" {
			hasGameMode = true
		}
	}
	if !hasGameMode {
		t.Error("JOGO deve conter jogos.game-mode")
	}
}
