package systemdiag

import (
	"context"
	"os/exec"
	"strings"

	"optimizer/internal/console"
)

// PowerPlan representa um plano de energia do Windows.
type PowerPlan struct {
	GUID  string `json:"guid"`
	Nome  string `json:"nome"`
	Ativo bool   `json:"ativo"`
}

// ListarPlanosEnergia lista os esquemas de energia instalados via powercfg.
func ListarPlanosEnergia(ctx context.Context) ([]PowerPlan, error) {
	cmd := exec.CommandContext(ctx, "powercfg", "/list")
	console.HideWindow(cmd)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var planos []PowerPlan
	linhas := strings.Split(string(out), "\n")
	for _, l := range linhas {
		l = strings.TrimSpace(l)
		if !strings.HasPrefix(l, "GUID do Esquema de Energia:") && !strings.HasPrefix(l, "Power Scheme GUID:") {
			continue
		}

		partes := strings.Split(l, ":")
		if len(partes) < 2 {
			continue
		}
		resto := strings.TrimSpace(partes[1])

		guid := ""
		nome := ""
		ativo := strings.Contains(l, "*")

		idxAbreParentese := strings.Index(resto, "(")
		idxFechaParentese := strings.LastIndex(resto, ")")
		if idxAbreParentese != -1 && idxFechaParentese != -1 {
			guid = strings.TrimSpace(resto[:idxAbreParentese])
			nome = strings.TrimSpace(resto[idxAbreParentese+1 : idxFechaParentese])
		} else {
			guid = resto
			nome = resto
		}

		planos = append(planos, PowerPlan{
			GUID:  guid,
			Nome:  nome,
			Ativo: ativo,
		})
	}
	return planos, nil
}

// AtivarPlanoEnergia define o plano de energia ativo via powercfg /setactive.
func AtivarPlanoEnergia(ctx context.Context, guid string) error {
	cmd := exec.CommandContext(ctx, "powercfg", "/setactive", guid)
	console.HideWindow(cmd)
	return cmd.Run()
}
