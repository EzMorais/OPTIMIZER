package main

import (
	"path/filepath"
	"testing"
)

func TestRunUsageAndHelp(t *testing.T) {
	cases := [][]string{
		{},
		{"ajuda"},
		{"help"},
		{"--help"},
		{"-h"},
	}

	for _, args := range cases {
		if err := run(args); err != nil {
			t.Errorf("run(%v) retornou erro inesperado: %v", args, err)
		}
	}
}

func TestRunComandoDesconhecido(t *testing.T) {
	err := run([]string{"inexistente"})
	if err == nil {
		t.Error("esperava erro para comando desconhecido")
	}
}

func TestRunListar(t *testing.T) {
	cases := [][]string{
		{"listar"},
		{"listar", "--detalhado"},
		{"listar", "--perfil", "pessoal"},
		{"listar", "--perfil", "trabalho"},
	}

	for _, args := range cases {
		if err := run(args); err != nil {
			t.Errorf("run(%v) falhou: %v", args, err)
		}
	}
}

func TestRunListarPerfilInvalido(t *testing.T) {
	err := run([]string{"listar", "--perfil", "invalido"})
	if err == nil {
		t.Error("esperava erro para perfil inválido")
	}
}

func TestRunPerfil(t *testing.T) {
	cases := [][]string{
		{"perfil"},
		{"perfil", "listar"},
		{"perfil", "aplicar", "jogo", "--simular"},
		{"perfil", "aplicar", "coding", "--simular"},
		{"perfil", "verificar", "jogo"},
		{"perfil", "verificar", "coding"},
		{"perfil", "restaurar", "jogo"},
		{"perfil", "aplicar", "network-fast", "--simular"},
		{"perfil", "aplicar", "network-dev", "--simular"},
		{"perfil", "aplicar", "network-remote", "--simular"},
		{"perfil", "aplicar", "network-presentation", "--simular"},
	}

	for _, args := range cases {
		if err := run(args); err != nil {
			t.Errorf("run(%v) falhou: %v", args, err)
		}
	}
}

func TestRunPerfilInexistente(t *testing.T) {
	err := run([]string{"perfil", "aplicar", "perfil-inexistente", "--simular"})
	if err == nil {
		t.Error("esperava erro para perfil inexistente")
	}
}

func TestRunAplicarSimulado(t *testing.T) {
	tempHist := filepath.Join(t.TempDir(), "test_hist.jsonl")
	err := run([]string{"aplicar", "visual.menu-show-delay", "--simular", "--historico", tempHist})
	if err != nil {
		t.Fatalf("run aplicar simulado falhou: %v", err)
	}
}

func TestRunHistoricoVazio(t *testing.T) {
	tempHist := filepath.Join(t.TempDir(), "vazio.jsonl")
	err := run([]string{"historico", "--historico", tempHist})
	if err != nil {
		t.Fatalf("run historico falhou: %v", err)
	}
}
