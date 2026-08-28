package netdiag

import (
	"context"
	"encoding/json"
	"fmt"
	"net/netip"
	"os/exec"
	"strconv"
	"strings"
)

// DNSAtual descreve os servidores DNS IPv4 usados pela interface que carrega
// a rota padrão atual.
type DNSAtual struct {
	InterfaceAlias string   `json:"interfaceAlias"`
	InterfaceIndex uint32   `json:"interfaceIndex"`
	Servidores     []string `json:"servidores"`
}

// DNSRunner é a fronteira mockável com os cmdlets nativos de DNS do Windows.
type DNSRunner interface {
	RunPowerShell(ctx context.Context, script string) ([]byte, error)
}

// LiveDNSRunner executa comandos PowerShell locais. A elevação é validada
// pela camada da aplicação antes de qualquer alteração.
type LiveDNSRunner struct{}

func (LiveDNSRunner) RunPowerShell(ctx context.Context, script string) ([]byte, error) {
	out, err := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-Command", script).CombinedOutput()
	if err != nil {
		return out, fmt.Errorf("PowerShell: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return out, nil
}

// LerDNSAtual encontra a interface IPv4 da rota padrão e seus resolvedores
// em uso. A leitura não altera nenhuma configuração.
func LerDNSAtual(ctx context.Context, runner DNSRunner) (DNSAtual, error) {
	const script = `$route = Get-NetRoute -AddressFamily IPv4 -DestinationPrefix '0.0.0.0/0' | Sort-Object RouteMetric, InterfaceMetric | Select-Object -First 1
if ($null -eq $route) { throw 'Nenhuma rota IPv4 padrão ativa foi encontrada.' }
$dns = Get-DnsClientServerAddress -InterfaceIndex $route.InterfaceIndex -AddressFamily IPv4 -ErrorAction Stop
[pscustomobject]@{ interfaceAlias = $route.InterfaceAlias; interfaceIndex = $route.InterfaceIndex; serverAddresses = @($dns.ServerAddresses) } | ConvertTo-Json -Compress`

	out, err := runner.RunPowerShell(ctx, script)
	if err != nil {
		return DNSAtual{}, err
	}
	var raw struct {
		InterfaceAlias  string   `json:"interfaceAlias"`
		InterfaceIndex  uint32   `json:"interfaceIndex"`
		ServerAddresses []string `json:"serverAddresses"`
	}
	if err := json.Unmarshal(out, &raw); err != nil {
		return DNSAtual{}, fmt.Errorf("lendo DNS atual: resposta inválida: %w", err)
	}
	if raw.InterfaceAlias == "" || raw.InterfaceIndex == 0 {
		return DNSAtual{}, fmt.Errorf("lendo DNS atual: interface ativa ausente")
	}
	return DNSAtual{
		InterfaceAlias: raw.InterfaceAlias,
		InterfaceIndex: raw.InterfaceIndex,
		Servidores:     deduplicarIPs(raw.ServerAddresses),
	}, nil
}

// ConfigurarDNS define explicitamente os servidores da interface ativa. Só
// aceita IPs literais, evitando que dados da interface virem comandos.
func ConfigurarDNS(ctx context.Context, runner DNSRunner, atual DNSAtual, servidores []string) error {
	if atual.InterfaceIndex == 0 {
		return fmt.Errorf("configurando DNS: interface ativa inválida")
	}
	servidores = deduplicarIPs(servidores)
	if len(servidores) == 0 {
		return fmt.Errorf("configurando DNS: informe ao menos um endereço IPv4 ou IPv6 válido")
	}
	quoted := make([]string, 0, len(servidores))
	for _, servidor := range servidores {
		quoted = append(quoted, "'"+servidor+"'")
	}
	script := "Set-DnsClientServerAddress -InterfaceIndex " + strconv.FormatUint(uint64(atual.InterfaceIndex), 10) +
		" -ServerAddresses @(" + strings.Join(quoted, ", ") + ") -ErrorAction Stop"
	if _, err := runner.RunPowerShell(ctx, script); err != nil {
		return fmt.Errorf("configurando DNS: %w", err)
	}
	return nil
}

func deduplicarIPs(servidores []string) []string {
	seen := make(map[string]bool)
	out := make([]string, 0, len(servidores))
	for _, servidor := range servidores {
		addr, err := netip.ParseAddr(strings.TrimSpace(servidor))
		if err != nil || !addr.IsValid() {
			continue
		}
		ip := addr.String()
		if !seen[ip] {
			seen[ip] = true
			out = append(out, ip)
		}
	}
	return out
}
