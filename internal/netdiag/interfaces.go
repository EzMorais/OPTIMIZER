package netdiag

import (
	"errors"
	"fmt"
	"net"
	"net/netip"
)

// Tipos de interface do Windows (IANA ifType) que interessam ao diagnóstico.
const (
	IfTypeOther    = 1
	IfTypeEthernet = 6
	IfTypePPP      = 23
	IfTypeLoopback = 24
	IfTypeWiFi     = 71
	IfTypeTunnel   = 131
)

// Interface é um adaptador de rede do ponto de vista do usuário.
type Interface struct {
	Index       uint32
	Luid        uint64
	Name        string // nome que aparece em "Conexões de Rede" (ex.: "Ethernet", "Wi-Fi")
	Description string // nome do driver/placa
	MTU         uint32
	Type        uint32
	Up          bool
	IPs         []netip.Addr
	Gateways    []netip.Addr
}

func (i Interface) KindLabel() string {
	switch i.Type {
	case IfTypeEthernet:
		return "cabo"
	case IfTypeWiFi:
		return "Wi-Fi"
	case IfTypePPP:
		return "PPP/PPPoE"
	case IfTypeTunnel:
		return "túnel/VPN"
	case IfTypeLoopback:
		return "loopback"
	}
	return "outro"
}

// LikelyTunnel indica adaptadores em que um MTU menor que 1500 costuma ser
// intencional — mexer neles é justamente onde o usuário se dá mal.
func (i Interface) LikelyTunnel() bool {
	return i.Type == IfTypeTunnel || i.Type == IfTypePPP
}

func (i Interface) HasIP(a netip.Addr) bool {
	for _, ip := range i.IPs {
		if ip == a {
			return true
		}
	}
	return false
}

// Resolve traduz nome ou IP em endereço IPv4.
func Resolve(host string) (netip.Addr, error) {
	if a, err := netip.ParseAddr(host); err == nil {
		if !a.Is4() {
			return netip.Addr{}, errors.New("a medição de MTU por enquanto só cobre IPv4")
		}
		return a, nil
	}
	ips, err := net.LookupIP(host)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("não foi possível resolver %q: %w", host, err)
	}
	for _, ip := range ips {
		if v4 := ip.To4(); v4 != nil {
			a, _ := netip.AddrFromSlice(v4)
			return a, nil
		}
	}
	return netip.Addr{}, fmt.Errorf("%q não tem endereço IPv4", host)
}

// OutboundIP descobre por qual IP local o Windows sairia para falar com target.
// O Dial em UDP não envia pacote nenhum — só faz o sistema escolher a rota.
func OutboundIP(target netip.Addr) (netip.Addr, error) {
	c, err := net.Dial("udp4", net.JoinHostPort(target.String(), "53"))
	if err != nil {
		return netip.Addr{}, fmt.Errorf("descobrindo a rota de saída: %w", err)
	}
	defer c.Close()

	ua, ok := c.LocalAddr().(*net.UDPAddr)
	if !ok {
		return netip.Addr{}, errors.New("endereço local inesperado")
	}
	a, ok := netip.AddrFromSlice(ua.IP.To4())
	if !ok {
		return netip.Addr{}, errors.New("endereço local não é IPv4")
	}
	return a, nil
}

// InterfaceForIP acha o adaptador dono de um IP local.
func InterfaceForIP(ifs []Interface, ip netip.Addr) *Interface {
	for i := range ifs {
		if ifs[i].HasIP(ip) {
			return &ifs[i]
		}
	}
	return nil
}

// OutboundInterface junta as duas coisas: por onde saímos para chegar em target.
func OutboundInterface(target netip.Addr) (*Interface, error) {
	ifs, err := Interfaces()
	if err != nil {
		return nil, err
	}
	local, err := OutboundIP(target)
	if err != nil {
		return nil, err
	}
	iface := InterfaceForIP(ifs, local)
	if iface == nil {
		return nil, fmt.Errorf("nenhum adaptador com o endereço local %s", local)
	}
	return iface, nil
}
