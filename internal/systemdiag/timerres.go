package systemdiag

import (
	"context"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"optimizer/internal/console"
)

// TimerResolutionInfo contém os dados de resolução do temporizador do kernel.
type TimerResolutionInfo struct {
	MinResolutionMs     float64 `json:"minResolutionMs"`
	MaxResolutionMs     float64 `json:"maxResolutionMs"`
	CurrentResolutionMs float64 `json:"currentResolutionMs"`
	IsHighPrecision     bool    `json:"isHighPrecision"`
	IsPersistent        bool    `json:"isPersistent"`
}

// SleepPrecisionResult contém a medição de desvio do timer do sistema.
type SleepPrecisionResult struct {
	TargetMs    float64   `json:"targetMs"`
	AverageMs   float64   `json:"averageMs"`
	MinMs       float64   `json:"minMs"`
	MaxMs       float64   `json:"maxMs"`
	StdDevMs    float64   `json:"stdDevMs"`
	JitterScore string    `json:"jitterScore"`
	Samples     []float64 `json:"samples"`
}

// TimerRunner define interface para gerenciar a tarefa de persistência do timer (mockável).
type TimerRunner interface {
	ExecCmd(ctx context.Context, name string, args ...string) ([]byte, error)
}

type defaultTimerRunner struct{}

func (r *defaultTimerRunner) ExecCmd(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	console.HideWindow(cmd)
	return cmd.CombinedOutput()
}

var currentTimerRunner TimerRunner = &defaultTimerRunner{}

// SetTimerRunner injeta um executor customizado em testes.
func SetTimerRunner(r TimerRunner) {
	currentTimerRunner = r
}

const taskNameTimer = "OptimizerTimerResolution"

// VerificarPersistenciaTimer checa se a tarefa agendada de 0.5ms está instalada.
func VerificarPersistenciaTimer(ctx context.Context) bool {
	out, err := currentTimerRunner.ExecCmd(ctx, "schtasks.exe", "/query", "/tn", taskNameTimer)
	if err != nil {
		return false
	}
	return strings.Contains(string(out), taskNameTimer)
}

// ConfigurarPersistenciaTimer ativa ou desativa o serviço de temporizador em segundo plano.
func ConfigurarPersistenciaTimer(ctx context.Context, ativar bool, desiredMs float64) error {
	if !ativar {
		_, _ = currentTimerRunner.ExecCmd(ctx, "schtasks.exe", "/delete", "/tn", taskNameTimer, "/f")
		_, _ = currentTimerRunner.ExecCmd(ctx, "taskkill.exe", "/f", "/im", "optimizerctl.exe", "/fi", "WINDOWTITLE eq OptimizerTimerDaemon*")
		return nil
	}

	if desiredMs <= 0 {
		desiredMs = 0.5
	}

	// Localiza o executável optimizerctl
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("não foi possível identificar o caminho do executável: %w", err)
	}
	dir := filepath.Dir(exePath)
	ctlPath := filepath.Join(dir, "optimizerctl.exe")

	if _, err := os.Stat(ctlPath); err != nil {
		// Fallback para diretório comum ou %LOCALAPPDATA%
		localApp := os.Getenv("LOCALAPPDATA")
		ctlFallback := filepath.Join(localApp, "Programs", "Optimizer", "optimizerctl.exe")
		if _, errF := os.Stat(ctlFallback); errF == nil {
			ctlPath = ctlFallback
		}
	}

	trArg := fmt.Sprintf(`"%s" timer-daemon --resolution %.3f`, ctlPath, desiredMs)

	// Cria tarefa agendada elevada no login do usuário
	out, err := currentTimerRunner.ExecCmd(ctx, "schtasks.exe", "/create", "/tn", taskNameTimer, "/tr", trArg, "/sc", "onlogon", "/rl", "highest", "/f")
	if err != nil {
		return fmt.Errorf("falha ao criar tarefa agendada: %w (%s)", err, strings.TrimSpace(string(out)))
	}

	// Executa a tarefa imediatamente
	_, _ = currentTimerRunner.ExecCmd(ctx, "schtasks.exe", "/run", "/tn", taskNameTimer)

	return nil
}

// MedirSleepPrecision executa um teste de jitter medindo a precisão do Sleep do SO.
func MedirSleepPrecision(samples int) SleepPrecisionResult {
	if samples < 5 {
		samples = 10
	}
	if samples > 100 {
		samples = 100
	}

	deltas := make([]float64, samples)
	target := 1.0 // 1 ms
	var sum float64
	minVal := math.MaxFloat64
	maxVal := 0.0

	for i := 0; i < samples; i++ {
		start := time.Now()
		time.Sleep(1 * time.Millisecond)
		elapsed := float64(time.Since(start).Nanoseconds()) / 1e6
		deltas[i] = elapsed
		sum += elapsed
		if elapsed < minVal {
			minVal = elapsed
		}
		if elapsed > maxVal {
			maxVal = elapsed
		}
	}

	avg := sum / float64(samples)
	var varianceSum float64
	for _, d := range deltas {
		varianceSum += (d - avg) * (d - avg)
	}
	stdDev := math.Sqrt(varianceSum / float64(samples))

	score := "Excelente (Ultra Baixo Jitter)"
	switch {
	case stdDev > 2.0:
		score = "Alto Jitter (Timer Impreciso)"
	case stdDev > 0.8:
		score = "Moderado (Padrão do Windows)"
	case stdDev > 0.3:
		score = "Bom"
	}

	return SleepPrecisionResult{
		TargetMs:    target,
		AverageMs:   math.Round(avg*1000) / 1000,
		MinMs:       math.Round(minVal*1000) / 1000,
		MaxMs:       math.Round(maxVal*1000) / 1000,
		StdDevMs:    math.Round(stdDev*1000) / 1000,
		JitterScore: score,
		Samples:     deltas,
	}
}
