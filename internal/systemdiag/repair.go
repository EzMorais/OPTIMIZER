package systemdiag

import (
	"context"
	"os/exec"
	"strings"
	"time"

	"optimizer/internal/console"
)

// RepairAuditReport contém os resultados de verificação de integridade não destrutiva.
type RepairAuditReport struct {
	Timestamp      time.Time `json:"timestamp"`
	DISMChecked    bool      `json:"dismChecked"`
	DISMHealthy    bool      `json:"dismHealthy"`
	DISMOutput     string    `json:"dismOutput"`
	SFCChecked     bool      `json:"sfcChecked"`
	SFCHealthy     bool      `json:"sfcHealthy"`
	SFCOutput      string    `json:"sfcOutput"`
	Interpretation string    `json:"interpretation"`
}

// ExecutarAuditoriaIntegridade executa verificações somente-leitura (DISM CheckHealth e SFC verifyonly).
func ExecutarAuditoriaIntegridade(ctx context.Context) RepairAuditReport {
	rep := RepairAuditReport{
		Timestamp:   time.Now(),
		DISMHealthy: true,
		SFCHealthy:  true,
	}

	// 1. DISM /Online /Cleanup-Image /CheckHealth (rápido e não destrutivo)
	cmdDISM := exec.CommandContext(ctx, "dism.exe", "/Online", "/Cleanup-Image", "/CheckHealth")
	console.HideWindow(cmdDISM)
	outDISM, err := cmdDISM.CombinedOutput()
	rep.DISMChecked = true
	rep.DISMOutput = strings.TrimSpace(string(outDISM))
	if err != nil || strings.Contains(strings.ToLower(rep.DISMOutput), "corrupted") || strings.Contains(strings.ToLower(rep.DISMOutput), "corrompido") {
		rep.DISMHealthy = false
	}

	// 2. SFC /verifyonly (somente verifica integridade sem alterar arquivos)
	cmdSFC := exec.CommandContext(ctx, "sfc.exe", "/verifyonly")
	console.HideWindow(cmdSFC)
	outSFC, errSFC := cmdSFC.CombinedOutput()
	rep.SFCChecked = true
	rep.SFCOutput = strings.TrimSpace(string(outSFC))
	if errSFC != nil || strings.Contains(strings.ToLower(rep.SFCOutput), "integrity violations") || strings.Contains(strings.ToLower(rep.SFCOutput), "violações de integridade") {
		rep.SFCHealthy = false
	}

	if rep.DISMHealthy && rep.SFCHealthy {
		rep.Interpretation = "A imagem do sistema e os arquivos protegidos do Windows estão íntegros e sem corrupção."
	} else {
		rep.Interpretation = "Foram detectadas inconsistências ou corrupções nos arquivos do sistema. Recomenda-se executar o reparo oficial do Windows."
	}

	return rep
}
