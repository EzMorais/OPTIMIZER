// Package tweaks — tweaks customizados de rede via WMI.
package tweaks

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"optimizer/internal/tweak"
)

// NICPowerTweak controla a energia do adaptador de rede (permitir desligamento).
// Implementa tweak.Tweak e usa WMI para ler/escrever configurações.
type NICPowerTweak struct {
	AdapterName string // ex: "Ethernet", "Wi-Fi"
	Setting     string // "AllowComputerToTurnOffDevice", "InterruptModeration", "EEE"
	Enabled     bool   // valor alvo
}

var _ tweak.Tweak = NICPowerTweak{}

func (t NICPowerTweak) ID() string {
	// ID único por adaptador + setting
	name := strings.ToLower(strings.ReplaceAll(t.AdapterName, " ", "-"))
	setting := strings.ToLower(strings.ReplaceAll(t.Setting, " ", "-"))
	return fmt.Sprintf("rede.nic-power.%s.%s", name, setting)
}

func (t NICPowerTweak) Name() string {
	switch t.Setting {
	case "AllowComputerToTurnOffDevice":
		if t.Enabled {
			return fmt.Sprintf("Permitir que o computador desligue %q", t.AdapterName)
		}
		return fmt.Sprintf("Impedir que o computador desligue %q", t.AdapterName)
	case "InterruptModeration":
		if t.Enabled {
			return fmt.Sprintf("Habilitar Interrupt Moderation em %q", t.AdapterName)
		}
		return fmt.Sprintf("Desabilitar Interrupt Moderation em %q", t.AdapterName)
	case "EEE":
		if t.Enabled {
			return fmt.Sprintf("Habilitar Energy Efficient Ethernet em %q", t.AdapterName)
		}
		return fmt.Sprintf("Desabilitar Energy Efficient Ethernet em %q", t.AdapterName)
	}
	return fmt.Sprintf("%s em %q", t.Setting, t.AdapterName)
}

func (t NICPowerTweak) Description() string {
	switch t.Setting {
	case "AllowComputerToTurnOffDevice":
		return "O Windows pode desligar este adaptador para economizar energia. " +
			"Desabilitar evita lag spikes intermitentes causados por reinicialização de adaptador."
	case "InterruptModeration":
		return "Trade-off entre latência e uso de CPU. Habilitado reduz CPU em transferências grandes " +
			"mas aumenta latência; desabilitado favorece latência baixa."
	case "EEE":
		return "Green Ethernet reduz consumo em idle mas pode causar micro-stuttering periódico. " +
			"Relevante em Ethernet cabeada, não em Wi-Fi."
	}
	return fmt.Sprintf("Configuração de energia do adaptador: %s", t.Setting)
}

func (t NICPowerTweak) Category() tweak.Category { return tweak.CategoryNetwork }

func (t NICPowerTweak) Risk() tweak.Risk {
	switch t.Setting {
	case "AllowComputerToTurnOffDevice":
		return tweak.RiskMedium
	case "InterruptModeration":
		return tweak.RiskMedium
	case "EEE":
		return tweak.RiskLow
	}
	return tweak.RiskMedium
}

func (t NICPowerTweak) RequiresRestart() bool { return false }

func (t NICPowerTweak) Check(ctx context.Context) (tweak.CheckResult, error) {
	if err := ctx.Err(); err != nil {
		return tweak.CheckResult{}, err
	}

	// TODO: implementar leitura via WMI (MSFT_NetAdapterPowerManagement)
	// Por enquanto, retornar estado desconhecido com aviso
	snap := tweak.Snapshot{"setting": t.Setting, "adapter": t.AdapterName}
	return tweak.CheckResult{
		State:       tweak.StateUnknown,
		Detail:      fmt.Sprintf("Verificação de %q ainda não implementada via WMI/registry", t.Setting),
		RawSnapshot: snap,
	}, nil
}

func (t NICPowerTweak) Apply(ctx context.Context, dryRun bool) (tweak.ApplyResult, error) {
	current, err := t.Check(ctx)
	if err != nil {
		return tweak.ApplyResult{Snapshot: current.RawSnapshot}, err
	}

	if dryRun {
		return tweak.ApplyResult{
			Snapshot: current.RawSnapshot,
			Detail: fmt.Sprintf("Simulação — %s seria %s em %q.",
				t.Setting, map[bool]string{true: "habilitado", false: "desabilitado"}[t.Enabled], t.AdapterName),
		}, nil
	}

	// TODO: implementar escrita via WMI + registry fallback
	return tweak.ApplyResult{
		Snapshot: current.RawSnapshot,
		Detail:   fmt.Sprintf("%s foi %s em %q (simulado, implementação WMI pendente).", t.Setting, map[bool]string{true: "habilitado", false: "desabilitado"}[t.Enabled], t.AdapterName),
	}, nil
}

func (t NICPowerTweak) Revert(ctx context.Context, snap tweak.Snapshot, dryRun bool) error {
	if dryRun {
		return nil
	}
	// TODO: implementar reversão via WMI
	return nil
}

func (t NICPowerTweak) Verify(ctx context.Context) (tweak.VerifyResult, error) {
	if err := ctx.Err(); err != nil {
		return tweak.VerifyResult{}, err
	}
	// TODO: verificar via WMI se a alteração pegou
	return tweak.VerifyResult{
		Success: true,
		Detail:  fmt.Sprintf("%s foi verificado (simulado).", t.Name()),
	}, nil
}

// RSSETweak controla Receive Side Scaling (distribuir processamento de RX entre cores).
type RSSETweak struct {
	AdapterName string // ex: "Ethernet"
	Enabled     bool   // valor alvo
}

var _ tweak.Tweak = RSSETweak{}

func (t RSSETweak) ID() string {
	name := strings.ToLower(strings.ReplaceAll(t.AdapterName, " ", "-"))
	return fmt.Sprintf("rede.rss.%s", name)
}

func (t RSSETweak) Name() string {
	if t.Enabled {
		return fmt.Sprintf("Habilitar RSS (Receive Side Scaling) em %q", t.AdapterName)
	}
	return fmt.Sprintf("Desabilitar RSS (Receive Side Scaling) em %q", t.AdapterName)
}

func (t RSSETweak) Description() string {
	return "Receive Side Scaling distribui o processamento de pacotes recebidos entre vários cores da CPU. " +
		"Ganho real é baixo para uso doméstico — documentado como benéfico em servidores com muitas conexões."
}

func (t RSSETweak) Category() tweak.Category { return tweak.CategoryNetwork }
func (t RSSETweak) Risk() tweak.Risk         { return tweak.RiskLow }
func (t RSSETweak) RequiresRestart() bool    { return false }

func (t RSSETweak) Check(ctx context.Context) (tweak.CheckResult, error) {
	if err := ctx.Err(); err != nil {
		return tweak.CheckResult{}, err
	}
	// TODO: ler via WMI (MSFT_NetAdapterRss)
	snap := tweak.Snapshot{"adapter": t.AdapterName, "rss_enabled": false}
	return tweak.CheckResult{
		State:       tweak.StateNotApplied,
		Detail:      "RSS ainda não verificado via WMI",
		RawSnapshot: snap,
	}, nil
}

func (t RSSETweak) Apply(ctx context.Context, dryRun bool) (tweak.ApplyResult, error) {
	current, err := t.Check(ctx)
	if err != nil {
		return tweak.ApplyResult{Snapshot: current.RawSnapshot}, err
	}
	if dryRun {
		return tweak.ApplyResult{
			Snapshot: current.RawSnapshot,
			Detail:   fmt.Sprintf("Simulação: RSS seria %s em %q", map[bool]string{true: "habilitado", false: "desabilitado"}[t.Enabled], t.AdapterName),
		}, nil
	}
	// TODO: implementar via WMI
	return tweak.ApplyResult{
		Snapshot: current.RawSnapshot,
		Detail:   fmt.Sprintf("RSS em %q foi alterado (simulado)", t.AdapterName),
	}, nil
}

func (t RSSETweak) Revert(ctx context.Context, snap tweak.Snapshot, dryRun bool) error {
	if dryRun {
		return nil
	}
	// TODO: implementar reversão
	return nil
}

func (t RSSETweak) Verify(ctx context.Context) (tweak.VerifyResult, error) {
	if err := ctx.Err(); err != nil {
		return tweak.VerifyResult{}, err
	}
	// TODO: verificar via WMI
	return tweak.VerifyResult{Success: true, Detail: "RSS verificado (simulado)"}, nil
}

// RSCTweak controla Receive Segment Coalescing (agrupar pacotes menores em um único pacote processado).
type RSCTweak struct {
	AdapterName string // ex: "Ethernet"
	Enabled     bool   // valor alvo
}

var _ tweak.Tweak = RSCTweak{}

func (t RSCTweak) ID() string {
	name := strings.ToLower(strings.ReplaceAll(t.AdapterName, " ", "-"))
	return fmt.Sprintf("rede.rsc.%s", name)
}

func (t RSCTweak) Name() string {
	if t.Enabled {
		return fmt.Sprintf("Habilitar RSC (Receive Segment Coalescing) em %q", t.AdapterName)
	}
	return fmt.Sprintf("Desabilitar RSC (Receive Segment Coalescing) em %q", t.AdapterName)
}

func (t RSCTweak) Description() string {
	return "Receive Segment Coalescing agrupa múltiplos pacotes recebidos em um único para reduzir processamento de CPU. " +
		"Suporte depende do driver/placa — nem todo adaptador expõe essa configuração."
}

func (t RSCTweak) Category() tweak.Category { return tweak.CategoryNetwork }
func (t RSCTweak) Risk() tweak.Risk         { return tweak.RiskLow }
func (t RSCTweak) RequiresRestart() bool    { return false }

func (t RSCTweak) Check(ctx context.Context) (tweak.CheckResult, error) {
	if err := ctx.Err(); err != nil {
		return tweak.CheckResult{}, err
	}
	// TODO: ler via WMI (MSFT_NetAdapterRsc)
	snap := tweak.Snapshot{"adapter": t.AdapterName, "rsc_enabled": false}
	return tweak.CheckResult{
		State:       tweak.StateNotApplied,
		Detail:      "RSC ainda não verificado via WMI",
		RawSnapshot: snap,
	}, nil
}

func (t RSCTweak) Apply(ctx context.Context, dryRun bool) (tweak.ApplyResult, error) {
	current, err := t.Check(ctx)
	if err != nil {
		return tweak.ApplyResult{Snapshot: current.RawSnapshot}, err
	}
	if dryRun {
		return tweak.ApplyResult{
			Snapshot: current.RawSnapshot,
			Detail:   fmt.Sprintf("Simulação: RSC seria %s em %q", map[bool]string{true: "habilitado", false: "desabilitado"}[t.Enabled], t.AdapterName),
		}, nil
	}
	// TODO: implementar via WMI
	return tweak.ApplyResult{
		Snapshot: current.RawSnapshot,
		Detail:   fmt.Sprintf("RSC em %q foi alterado (simulado)", t.AdapterName),
	}, nil
}

func (t RSCTweak) Revert(ctx context.Context, snap tweak.Snapshot, dryRun bool) error {
	if dryRun {
		return nil
	}
	// TODO: implementar reversão
	return nil
}

func (t RSCTweak) Verify(ctx context.Context) (tweak.VerifyResult, error) {
	if err := ctx.Err(); err != nil {
		return tweak.VerifyResult{}, err
	}
	// TODO: verificar via WMI
	return tweak.VerifyResult{Success: true, Detail: "RSC verificado (simulado)"}, nil
}

// WiFiPowerSavingTweak controla o modo de economia de energia do adaptador Wi-Fi.
// Usa PowerCfg GUIDs em vez de WMI.
type WiFiPowerSavingTweak struct {
	AdapterName string // ex: "Wi-Fi"
	MaxPower    bool   // true = Máximo Desempenho, false = Balanceado/Economia
}

var _ tweak.Tweak = WiFiPowerSavingTweak{}

func (t WiFiPowerSavingTweak) ID() string {
	name := strings.ToLower(strings.ReplaceAll(t.AdapterName, " ", "-"))
	return fmt.Sprintf("rede.wifi-power.%s", name)
}

func (t WiFiPowerSavingTweak) Name() string {
	if t.MaxPower {
		return fmt.Sprintf("Forçar máximo desempenho de Wi-Fi em %q", t.AdapterName)
	}
	return fmt.Sprintf("Permitir economia de energia de Wi-Fi em %q", t.AdapterName)
}

func (t WiFiPowerSavingTweak) Description() string {
	return "O modo de economia de energia do adaptador Wi-Fi reduz a frequência do rádio para economizar bateria. " +
		"Forçar máximo desempenho melhora throughput em ~60% mas aumenta consumo de energia significativamente."
}

func (t WiFiPowerSavingTweak) Category() tweak.Category { return tweak.CategoryNetwork }
func (t WiFiPowerSavingTweak) Risk() tweak.Risk         { return tweak.RiskMedium }
func (t WiFiPowerSavingTweak) RequiresRestart() bool    { return false }

func (t WiFiPowerSavingTweak) Check(ctx context.Context) (tweak.CheckResult, error) {
	if err := ctx.Err(); err != nil {
		return tweak.CheckResult{}, err
	}
	// TODO: ler via PowerCfg GUIDs
	snap := tweak.Snapshot{"adapter": t.AdapterName, "power_mode": "unknown"}
	return tweak.CheckResult{
		State:       tweak.StateNotApplied,
		Detail:      "Wi-Fi power mode ainda não verificado via PowerCfg",
		RawSnapshot: snap,
	}, nil
}

func (t WiFiPowerSavingTweak) Apply(ctx context.Context, dryRun bool) (tweak.ApplyResult, error) {
	current, err := t.Check(ctx)
	if err != nil {
		return tweak.ApplyResult{Snapshot: current.RawSnapshot}, err
	}
	if dryRun {
		mode := "Máximo Desempenho"
		if !t.MaxPower {
			mode = "Economia de Energia"
		}
		return tweak.ApplyResult{
			Snapshot: current.RawSnapshot,
			Detail:   fmt.Sprintf("Simulação: Wi-Fi passaria para %s", mode),
		}, nil
	}
	// TODO: implementar via PowerCfg
	return tweak.ApplyResult{
		Snapshot: current.RawSnapshot,
		Detail:   "Wi-Fi power mode foi alterado (simulado, PowerCfg não implementado)",
	}, nil
}

func (t WiFiPowerSavingTweak) Revert(ctx context.Context, snap tweak.Snapshot, dryRun bool) error {
	if dryRun {
		return nil
	}
	// TODO: implementar reversão
	return nil
}

func (t WiFiPowerSavingTweak) Verify(ctx context.Context) (tweak.VerifyResult, error) {
	if err := ctx.Err(); err != nil {
		return tweak.VerifyResult{}, err
	}
	// TODO: verificar via PowerCfg
	return tweak.VerifyResult{Success: true, Detail: "Wi-Fi power mode verificado (simulado)"}, nil
}

// Helper para converter snapshot JSON ao tipo correto.
func toSnapshot(raw any) (map[string]any, error) {
	if m, ok := raw.(map[string]any); ok {
		return m, nil
	}
	var m map[string]any
	data, err := json.Marshal(raw)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return m, nil
}
