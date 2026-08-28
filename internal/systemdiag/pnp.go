package systemdiag

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// PnpDevice representa um dispositivo Plug-and-Play no Windows.
type PnpDevice struct {
	InstanceID   string `json:"InstanceId"`
	FriendlyName string `json:"FriendlyName"`
	Class        string `json:"Class"`
	Status       string `json:"Status"`
	Present      bool   `json:"Present"`
}

// PnpRunner define a interface de execução de comandos para dispositivos PnP (mockável em testes).
type PnpRunner interface {
	ExecPowerShell(ctx context.Context, script string) ([]byte, error)
	RemoveDevice(ctx context.Context, instanceID string) error
}

type defaultPnpRunner struct{}

func (r *defaultPnpRunner) ExecPowerShell(ctx context.Context, script string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-Command", script)
	return cmd.Output()
}

func (r *defaultPnpRunner) RemoveDevice(ctx context.Context, instanceID string) error {
	cmd := exec.CommandContext(ctx, "pnputil.exe", "/remove-device", instanceID)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("pnputil erro ao remover %s: %v (%s)", instanceID, err, strings.TrimSpace(string(out)))
	}
	return nil
}

var currentPnpRunner PnpRunner = &defaultPnpRunner{}

// SetPnpRunner permite injetar um executor alternativo para testes.
func SetPnpRunner(r PnpRunner) {
	currentPnpRunner = r
}

// ListarDispositivosFantasmas identifica dispositivos PnP desconectados ou órfãos (status Unknown ou Present=false).
func ListarDispositivosFantasmas(ctx context.Context) ([]PnpDevice, error) {
	psScript := `Get-PnpDevice | Where-Object { ($_.Present -eq $false -or $_.Status -eq 'Unknown') -and $_.InstanceId } | Select-Object InstanceId, FriendlyName, Class, Status, Present | ConvertTo-Json -Compress`
	out, err := currentPnpRunner.ExecPowerShell(ctx, psScript)
	if err != nil {
		return nil, fmt.Errorf("falha ao consultar dispositivos PnP: %w", err)
	}

	trimmed := strings.TrimSpace(string(out))
	if trimmed == "" || trimmed == "null" {
		return []PnpDevice{}, nil
	}

	// PowerShell ConvertTo-Json pode retornar um único objeto se for 1 elemento ou array se múltiplos
	if strings.HasPrefix(trimmed, "{") {
		var single PnpDevice
		if err := json.Unmarshal([]byte(trimmed), &single); err != nil {
			return nil, fmt.Errorf("falha ao decodificar dispositivo PnP: %w", err)
		}
		if single.FriendlyName == "" {
			single.FriendlyName = single.InstanceID
		}
		return []PnpDevice{single}, nil
	}

	var list []PnpDevice
	if err := json.Unmarshal([]byte(trimmed), &list); err != nil {
		return nil, fmt.Errorf("falha ao decodificar lista de dispositivos PnP: %w", err)
	}
	for i := range list {
		if list[i].FriendlyName == "" {
			list[i].FriendlyName = list[i].InstanceID
		}
	}
	return list, nil
}

// LimparDispositivosFantasmas remove os dispositivos informados usando pnputil.
func LimparDispositivosFantasmas(ctx context.Context, instanceIDs []string) (int, []string, error) {
	removidos := 0
	erros := make([]string, 0)

	for _, id := range instanceIDs {
		trimmed := strings.TrimSpace(id)
		if trimmed == "" {
			continue
		}
		if err := currentPnpRunner.RemoveDevice(ctx, trimmed); err != nil {
			erros = append(erros, err.Error())
		} else {
			removidos++
		}
	}

	return removidos, erros, nil
}
