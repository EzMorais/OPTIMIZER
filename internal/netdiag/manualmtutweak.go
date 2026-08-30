package netdiag

import "fmt"

// NewManualMTUTweak cria um ajuste editável, mas ainda sujeito à validação do
// mesmo controlador e ao histórico transacional usado pelo medidor automático.
func NewManualMTUTweak(iface Interface, mtu uint32, ctl MTUController) (MTUTweak, error) {
	validated, err := ValidateNetworkConfig(NetworkConfig{MTU: mtu})
	if err != nil {
		return MTUTweak{}, err
	}
	return MTUTweak{Iface: iface, Target: validated.MTU, Ctl: ctl, Host: "configuração manual"}, nil
}

// ManualMTUDescription evita que o editor prometa que MSS é uma configuração
// independente: os valores exibidos são derivados do MTU escolhido.
func ManualMTUDescription(mtu uint32) string {
	v, err := ValidateNetworkConfig(NetworkConfig{MTU: mtu})
	if err != nil {
		return err.Error()
	}
	return fmt.Sprintf("MTU %d; MSS de referência IPv4 %d e IPv6 %d. O MSS real é negociado por conexão.", v.MTU, v.MSSIPv4, v.MSSIPv6)
}
