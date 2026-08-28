package netdiag

import (
	"context"
	"strings"
	"testing"
)

type fakeDNSRunner struct {
	outputs [][]byte
	scripts []string
}

func (f *fakeDNSRunner) RunPowerShell(_ context.Context, script string) ([]byte, error) {
	f.scripts = append(f.scripts, script)
	if len(f.outputs) == 0 {
		return nil, nil
	}
	out := f.outputs[0]
	f.outputs = f.outputs[1:]
	return out, nil
}

func TestLeEConfiguraDNSDaInterfaceAtiva(t *testing.T) {
	runner := &fakeDNSRunner{outputs: [][]byte{[]byte(`{"interfaceAlias":"Ethernet","interfaceIndex":12,"serverAddresses":["192.168.1.1"]}`)}}

	atual, err := LerDNSAtual(context.Background(), runner)
	if err != nil {
		t.Fatalf("LerDNSAtual: %v", err)
	}
	if atual.InterfaceAlias != "Ethernet" || atual.InterfaceIndex != 12 {
		t.Fatalf("interface inesperada: %+v", atual)
	}
	if len(atual.Servidores) != 1 || atual.Servidores[0] != "192.168.1.1" {
		t.Fatalf("servidores atuais inesperados: %+v", atual.Servidores)
	}

	if err := ConfigurarDNS(context.Background(), runner, atual, []string{"1.1.1.1", "1.0.0.1"}); err != nil {
		t.Fatalf("ConfigurarDNS: %v", err)
	}
	if len(runner.scripts) != 2 {
		t.Fatalf("esperado leitura e gravação, obteve %d chamadas", len(runner.scripts))
	}
	if !strings.Contains(runner.scripts[1], "Set-DnsClientServerAddress") || !strings.Contains(runner.scripts[1], "1.1.1.1") {
		t.Errorf("script de configuração não recebeu os DNS escolhidos: %s", runner.scripts[1])
	}
}

func TestConfigurarDNSRecusaEnderecoInvalido(t *testing.T) {
	runner := &fakeDNSRunner{}
	err := ConfigurarDNS(context.Background(), runner, DNSAtual{InterfaceAlias: "Ethernet"}, []string{"dns.invalido"})
	if err == nil {
		t.Fatal("esperado erro para servidor DNS inválido")
	}
	if len(runner.scripts) != 0 {
		t.Fatalf("nenhum comando deveria ser executado, obteve %d", len(runner.scripts))
	}
}
