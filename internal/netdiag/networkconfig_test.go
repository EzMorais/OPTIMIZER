package netdiag

import "testing"

func TestValidateNetworkConfigCalculaMSS(t *testing.T) {
	cfg := NetworkConfig{MTU: 1492}
	got, err := ValidateNetworkConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if got.MSSIPv4 != 1452 || got.MSSIPv6 != 1432 {
		t.Fatalf("MSS = %d/%d, queria 1452/1432", got.MSSIPv4, got.MSSIPv6)
	}
}

func TestValidateNetworkConfigRejeitaMTUInvalido(t *testing.T) {
	if _, err := ValidateNetworkConfig(NetworkConfig{MTU: 500}); err == nil {
		t.Fatal("MTU inválido foi aceito")
	}
}

func TestValidateNetworkConfigNaoAceitaMSSForcado(t *testing.T) {
	if _, err := ValidateNetworkConfig(NetworkConfig{MTU: 1500, MSSIPv4: 1200}); err == nil {
		t.Fatal("MSS inconsistente foi aceito")
	}
}
