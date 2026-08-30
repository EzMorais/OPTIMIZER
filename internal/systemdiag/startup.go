// Package systemdiag gerencia itens de inicialização, planos de energia e diagnósticos do sistema.
package systemdiag

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"optimizer/internal/winreg"
)

// StartupItem representa um programa configurado para iniciar junto com o Windows.
type StartupItem struct {
	ID        string `json:"id"`
	Nome      string `json:"nome"`
	Comando   string `json:"comando"`
	Origem    string `json:"origem"` // "HKCU", "HKLM", "Pasta Inicializar"
	Ativo     bool   `json:"ativo"`
	UserScope bool   `json:"userScope"`
}

const (
	regRun = `Software\Microsoft\Windows\CurrentVersion\Run`
)

// ListarStartupItems lista programas na inicialização do HKCU e HKLM.
func ListarStartupItems(b winreg.Backend) ([]StartupItem, error) {
	return ListarStartupItemsContext(context.Background(), b)
}

// ListarStartupItemsContext aborta entre leituras quando a janela da UI expira.
func ListarStartupItemsContext(ctx context.Context, b winreg.Backend) ([]StartupItem, error) {
	var itens []StartupItem
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// 1. HKCU Run
	nomesHKCU, err := b.ValueNames(winreg.HKCU, regRun)
	if err == nil {
		for _, n := range nomesHKCU {
			if err := ctx.Err(); err != nil {
				return itens, err
			}
			cmd, err := b.GetString(winreg.HKCU, regRun, n)
			if err == nil && strings.TrimSpace(cmd) != "" {
				itens = append(itens, StartupItem{
					ID:        "hkcu." + n,
					Nome:      n,
					Comando:   cmd,
					Origem:    "HKCU (Usuário)",
					Ativo:     true,
					UserScope: true,
				})
			}
		}
	}

	// 2. HKLM Run
	nomesHKLM, err := b.ValueNames(winreg.HKLM, regRun)
	if err == nil {
		for _, n := range nomesHKLM {
			if err := ctx.Err(); err != nil {
				return itens, err
			}
			cmd, err := b.GetString(winreg.HKLM, regRun, n)
			if err == nil && strings.TrimSpace(cmd) != "" {
				itens = append(itens, StartupItem{
					ID:        "hklm." + n,
					Nome:      n,
					Comando:   cmd,
					Origem:    "HKLM (Máquina)",
					Ativo:     true,
					UserScope: false,
				})
			}
		}
	}

	// 3. Pasta de Inicialização (%APPDATA%\Microsoft\Windows\Start Menu\Programs\Startup)
	appData := os.Getenv("APPDATA")
	if appData != "" {
		startupDir := filepath.Join(appData, "Microsoft", "Windows", "Start Menu", "Programs", "Startup")
		entries, _ := os.ReadDir(startupDir)
		for _, e := range entries {
			if err := ctx.Err(); err != nil {
				return itens, err
			}
			if !e.IsDir() && !strings.HasSuffix(strings.ToLower(e.Name()), ".ini") {
				itens = append(itens, StartupItem{
					ID:        "folder." + e.Name(),
					Nome:      e.Name(),
					Comando:   filepath.Join(startupDir, e.Name()),
					Origem:    "Pasta Inicializar",
					Ativo:     true,
					UserScope: true,
				})
			}
		}
	}

	return itens, nil
}

// AlternarStartup ativa ou desativa um item de inicialização com segurança.
func AlternarStartup(b winreg.Backend, id string, ativar bool) error {
	partes := strings.SplitN(id, ".", 2)
	if len(partes) != 2 {
		return fmt.Errorf("id de inicialização inválido: %s", id)
	}
	tipo, nome := partes[0], partes[1]

	hive := winreg.HKCU
	if tipo == "hklm" {
		hive = winreg.HKLM
	}

	if !ativar {
		return b.DeleteValue(hive, regRun, nome)
	}
	return nil
}
