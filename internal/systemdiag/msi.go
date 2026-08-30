package systemdiag

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"optimizer/internal/console"
	"optimizer/internal/winreg"
)

// DispositivoMSI representa um hardware PCIe (GPU, Rede, USB) e seu status de modo MSI.
type DispositivoMSI struct {
	ID              string `json:"id"`
	Nome            string `json:"nome"`
	Classe          string `json:"classe"`
	CaminhoRegistro string `json:"caminhoRegistro"`
	MSISupported    bool   `json:"msiSupported"`
	StatusRotulo    string `json:"statusRotulo"`
}

// MsiRunner define a interface de execução para consulta e alteração de MSI.
type MsiRunner interface {
	ExecPowerShell(ctx context.Context, script string) ([]byte, error)
}

type defaultMsiRunner struct{}

func (r *defaultMsiRunner) ExecPowerShell(ctx context.Context, script string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-Command", script)
	console.HideWindow(cmd)
	return cmd.Output()
}

var currentMsiRunner MsiRunner = &defaultMsiRunner{}

// SetMsiRunner permite injetar executor alternativo para testes.
func SetMsiRunner(r MsiRunner) {
	currentMsiRunner = r
}

type msiDeviceRaw struct {
	InstanceID   string `json:"InstanceId"`
	FriendlyName string `json:"FriendlyName"`
	Class        string `json:"Class"`
	MSISupported int    `json:"MSISupported"`
	RegPath      string `json:"RegPath"`
}

// ListarDispositivosMSI varre os dispositivos PCI (GPU, Rede, USB) e reporta suporte e status MSI.
func ListarDispositivosMSI(ctx context.Context) ([]DispositivoMSI, error) {
	psScript := `Get-PnpDevice -Class 'Display','Net','USB' -ErrorAction SilentlyContinue | Where-Object { $_.InstanceId -like 'PCI*' } | ForEach-Object {
		$path = "HKLM:\SYSTEM\CurrentControlSet\Enum\$($_.InstanceId)\Device Parameters\Interrupt Management\MessageSignaledInterruptProperties"
		$msi = 0
		if (Test-Path $path) {
			$val = Get-ItemProperty -Path $path -Name MSISupported -ErrorAction SilentlyContinue
			if ($val -and $val.MSISupported -eq 1) { $msi = 1 }
		}
		[PSCustomObject]@{
			InstanceId = $_.InstanceId
			FriendlyName = $_.FriendlyName
			Class = $_.Class
			MSISupported = $msi
			RegPath = "SYSTEM\CurrentControlSet\Enum\$($_.InstanceId)\Device Parameters\Interrupt Management\MessageSignaledInterruptProperties"
		}
	} | ConvertTo-Json -Compress`

	out, err := currentMsiRunner.ExecPowerShell(ctx, psScript)
	if err != nil {
		return nil, fmt.Errorf("falha ao consultar dispositivos MSI: %w", err)
	}

	trimmed := strings.TrimSpace(string(out))
	if trimmed == "" || trimmed == "null" {
		return []DispositivoMSI{}, nil
	}

	var rawList []msiDeviceRaw
	if strings.HasPrefix(trimmed, "[") {
		if err := json.Unmarshal([]byte(trimmed), &rawList); err != nil {
			return nil, fmt.Errorf("erro ao decodificar lista de dispositivos MSI: %w", err)
		}
	} else {
		var single msiDeviceRaw
		if err := json.Unmarshal([]byte(trimmed), &single); err != nil {
			return nil, fmt.Errorf("erro ao decodificar dispositivo MSI: %w", err)
		}
		rawList = append(rawList, single)
	}

	var resultado []DispositivoMSI
	for _, r := range rawList {
		nome := r.FriendlyName
		if nome == "" {
			nome = r.InstanceID
		}
		msiOn := r.MSISupported == 1
		status := "IRQ Clássico (Line-Based)"
		if msiOn {
			status = "MSI Mode Ativo"
		}

		resultado = append(resultado, DispositivoMSI{
			ID:              r.InstanceID,
			Nome:            nome,
			Classe:          r.Class,
			CaminhoRegistro: r.RegPath,
			MSISupported:    msiOn,
			StatusRotulo:    status,
		})
	}

	return resultado, nil
}

// AlternarModoMSI ativa ou desativa o modo MSI para um dispositivo específico no registro.
func AlternarModoMSI(b winreg.Backend, caminhoRegistro string, ativar bool) error {
	if b == nil {
		return fmt.Errorf("backend de registro nulo")
	}
	val := uint32(0)
	if ativar {
		val = 1
	}
	return b.SetDWord(winreg.HKLM, caminhoRegistro, "MSISupported", val)
}
