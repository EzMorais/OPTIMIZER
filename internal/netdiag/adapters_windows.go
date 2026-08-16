//go:build windows

package netdiag

import (
	"fmt"
	"net/netip"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Interfaces lista os adaptadores IPv4 da máquina com o MTU configurado.
func Interfaces() ([]Interface, error) {
	const flags = windows.GAA_FLAG_INCLUDE_GATEWAYS |
		windows.GAA_FLAG_SKIP_ANYCAST |
		windows.GAA_FLAG_SKIP_MULTICAST |
		windows.GAA_FLAG_SKIP_DNS_SERVER

	size := uint32(15000)
	var buf []byte
	for tries := 0; tries < 4; tries++ {
		buf = make([]byte, size)
		err := windows.GetAdaptersAddresses(windows.AF_INET, flags, 0,
			(*windows.IpAdapterAddresses)(unsafe.Pointer(&buf[0])), &size)
		if err == nil {
			break
		}
		if err == windows.ERROR_BUFFER_OVERFLOW {
			continue // size já foi atualizado com o necessário
		}
		return nil, fmt.Errorf("listando adaptadores de rede: %w", err)
	}

	var out []Interface
	for aa := (*windows.IpAdapterAddresses)(unsafe.Pointer(&buf[0])); aa != nil; aa = aa.Next {
		iface := Interface{
			Index:       aa.IfIndex,
			Luid:        aa.Luid,
			Name:        windows.UTF16PtrToString(aa.FriendlyName),
			Description: windows.UTF16PtrToString(aa.Description),
			MTU:         aa.Mtu,
			Type:        aa.IfType,
			Up:          aa.OperStatus == windows.IfOperStatusUp,
		}
		for ua := aa.FirstUnicastAddress; ua != nil; ua = ua.Next {
			if a, ok := addrFrom(&ua.Address); ok {
				iface.IPs = append(iface.IPs, a)
			}
		}
		for ga := aa.FirstGatewayAddress; ga != nil; ga = ga.Next {
			if a, ok := addrFrom(&ga.Address); ok {
				iface.Gateways = append(iface.Gateways, a)
			}
		}
		out = append(out, iface)
	}
	return out, nil
}

func addrFrom(sa *windows.SocketAddress) (netip.Addr, bool) {
	ip := sa.IP()
	if ip == nil {
		return netip.Addr{}, false
	}
	v4 := ip.To4()
	if v4 == nil {
		return netip.Addr{}, false
	}
	return netip.AddrFromSlice(v4)
}

// GetMTU lê o MTU IPv4 atual de um adaptador pelo LUID.
func GetMTU(luid uint64) (uint32, error) {
	row := windows.MibIpInterfaceRow{Family: windows.AF_INET, InterfaceLuid: luid}
	if err := windows.GetIpInterfaceEntry(&row); err != nil {
		return 0, fmt.Errorf("lendo configuração IPv4 do adaptador: %w", err)
	}
	return row.NlMtu, nil
}

// SetMTU grava o MTU IPv4 do adaptador via SetIpInterfaceEntry — a API pública
// equivalente ao `netsh interface ipv4 set subinterface ... mtu=`. Exige
// privilégio de administrador.
//
// Nota de engenharia: a persistência entre reinícios precisa ser confirmada em
// VM antes do lançamento (docs/arquitetura-app-desktop.md, seção 7) — se não
// persistir, o app reaplica na inicialização em vez de prometer o que não faz.
func SetMTU(luid uint64, mtu uint32) error {
	if mtu < MinWindowsIPMTU || mtu > 9000 {
		return fmt.Errorf("MTU fora da faixa aceita pelo Windows (%d a 9000): %d", MinWindowsIPMTU, mtu)
	}
	if err := procSetIPInterface.Find(); err != nil {
		return fmt.Errorf("iphlpapi.dll não expôs SetIpInterfaceEntry: %w", err)
	}

	row := windows.MibIpInterfaceRow{Family: windows.AF_INET, InterfaceLuid: luid}
	if err := windows.GetIpInterfaceEntry(&row); err != nil {
		return fmt.Errorf("lendo configuração IPv4 do adaptador: %w", err)
	}
	row.NlMtu = mtu
	// Exigência documentada da Microsoft: para IPv4, SitePrefixLength precisa
	// ser 0 na escrita, senão a chamada falha com ERROR_INVALID_PARAMETER.
	row.SitePrefixLength = 0

	r1, _, _ := syscall.SyscallN(procSetIPInterface.Addr(), uintptr(unsafe.Pointer(&row)))
	if r1 != 0 {
		errno := syscall.Errno(r1)
		if errno == syscall.ERROR_ACCESS_DENIED {
			return fmt.Errorf("alterar o MTU exige permissão de administrador")
		}
		return fmt.Errorf("gravando MTU: %w", errno)
	}
	return nil
}

// LiveMTU é o controlador de MTU que fala com o Windows de verdade.
type LiveMTU struct{}

var _ MTUController = LiveMTU{}

func (LiveMTU) Get(luid uint64) (uint32, error)   { return GetMTU(luid) }
func (LiveMTU) Set(luid uint64, mtu uint32) error { return SetMTU(luid, mtu) }
