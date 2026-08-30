package netdiag

import "fmt"

// NetworkConfig representa somente parâmetros de rede que o app consegue
// validar e reverter. MSS não é uma chave global confiável no Windows: ele é
// negociado por conexão, portanto é derivado do MTU e exibido como referência.
type NetworkConfig struct {
	MTU     uint32
	MSSIPv4 uint32
	MSSIPv6 uint32
}

type ValidatedNetworkConfig struct {
	MTU     uint32
	MSSIPv4 uint32
	MSSIPv6 uint32
}

func ValidateNetworkConfig(cfg NetworkConfig) (ValidatedNetworkConfig, error) {
	if cfg.MTU < MinWindowsIPMTU || cfg.MTU > 9000 {
		return ValidatedNetworkConfig{}, fmt.Errorf("MTU fora da faixa aceita pelo Windows (%d a 9000): %d", MinWindowsIPMTU, cfg.MTU)
	}
	v := ValidatedNetworkConfig{
		MTU:     cfg.MTU,
		MSSIPv4: cfg.MTU - 40,
		MSSIPv6: cfg.MTU - 60,
	}
	if cfg.MSSIPv4 != 0 && cfg.MSSIPv4 != v.MSSIPv4 {
		return ValidatedNetworkConfig{}, fmt.Errorf("MSS IPv4 deve ser MTU - 40 (%d)", v.MSSIPv4)
	}
	if cfg.MSSIPv6 != 0 && cfg.MSSIPv6 != v.MSSIPv6 {
		return ValidatedNetworkConfig{}, fmt.Errorf("MSS IPv6 deve ser MTU - 60 (%d)", v.MSSIPv6)
	}
	return v, nil
}
