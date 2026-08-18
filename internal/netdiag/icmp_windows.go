//go:build windows

package netdiag

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"net/netip"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// A medição usa IcmpSendEcho (iphlpapi.dll) em vez de chamar ping.exe.
// Motivo, alinhado com docs/arquitetura-app-desktop.md, seção 2: API direta não
// paga o custo de criar processo, não depende do idioma do Windows para
// interpretar o resultado (o texto do ping é traduzido) e não parece "programa
// comum executando ferramenta de linha de comando escondido", que é heurística
// clássica de antivírus. Diferente de sockets ICMP crus, esta API NÃO exige
// privilégio de administrador.
var (
	modiphlpapi         = windows.NewLazySystemDLL("iphlpapi.dll")
	procIcmpCreateFile  = modiphlpapi.NewProc("IcmpCreateFile")
	procIcmpCloseHandle = modiphlpapi.NewProc("IcmpCloseHandle")
	procIcmpSendEcho    = modiphlpapi.NewProc("IcmpSendEcho")
	procSetIPInterface  = modiphlpapi.NewProc("SetIpInterfaceEntry")
)

// IP_FLAG_DF — "don't fragment". É o mesmo sinalizador que o `-f` do ping liga.
const ipFlagDF = 0x02

// Códigos IP_STATUS relevantes (winerror/ipexport.h).
const (
	ipSuccess              = 0
	ipDestNetUnreachable   = 11002
	ipDestHostUnreachable  = 11003
	ipDestProtUnreachable  = 11004
	ipDestPortUnreachable  = 11005
	ipPacketTooBig         = 11009
	ipReqTimedOut          = 11010
	ipBadRoute             = 11012
	ipTTLExpiredTransit    = 11013
	ipTTLExpiredReassembly = 11014
	ipGeneralFailure       = 11050
)

type ipOptionInformation struct {
	TTL         uint8
	TOS         uint8
	Flags       uint8
	OptionsSize uint8
	OptionsData *byte
}

type icmpEchoReply struct {
	Address       uint32
	Status        uint32
	RoundTripTime uint32
	DataSize      uint16
	Reserved      uint16
	Data          *byte
	Options       ipOptionInformation
}

// LiveProber manda eco ICMP de verdade.
type LiveProber struct{}

var _ Prober = LiveProber{}

func statusFromIP(code uint32) (ProbeStatus, error) {
	switch code {
	case ipSuccess:
		return ProbeOK, nil
	case ipPacketTooBig:
		return ProbeTooBig, nil
	case ipReqTimedOut:
		return ProbeTimeout, nil
	case ipDestNetUnreachable, ipDestHostUnreachable, ipDestProtUnreachable,
		ipDestPortUnreachable, ipBadRoute:
		return ProbeUnreachable, nil
	case ipTTLExpiredTransit, ipTTLExpiredReassembly:
		// Não é resposta do destino, mas também não é "pacote grande demais".
		return ProbeTimeout, nil
	default:
		return ProbeTimeout, fmt.Errorf("erro de ICMP do Windows (código %d)", code)
	}
}

func (LiveProber) Ping(ctx context.Context, dst netip.Addr, payload int, timeout time.Duration) (ProbeStatus, error) {
	if err := ctx.Err(); err != nil {
		return ProbeTimeout, err
	}
	if !dst.Is4() {
		return ProbeUnreachable, errors.New("a medição de MTU por enquanto só cobre IPv4")
	}
	if payload <= 0 {
		return ProbeOK, errors.New("tamanho de teste inválido")
	}
	for _, p := range []*windows.LazyProc{procIcmpCreateFile, procIcmpCloseHandle, procIcmpSendEcho} {
		if err := p.Find(); err != nil {
			return ProbeTimeout, fmt.Errorf("iphlpapi.dll não expôs %s: %w", p.Name, err)
		}
	}

	handle, _, lastErr := syscall.SyscallN(procIcmpCreateFile.Addr())
	if handle == uintptr(windows.InvalidHandle) || handle == 0 {
		return ProbeTimeout, fmt.Errorf("abrindo canal ICMP: %w", lastErr)
	}
	defer syscall.SyscallN(procIcmpCloseHandle.Addr(), handle)

	b4 := dst.As4()
	// IPAddr é o in_addr do Windows: os 4 bytes na ordem a.b.c.d em memória.
	dest := binary.LittleEndian.Uint32(b4[:])

	data := make([]byte, payload)
	for i := range data {
		data[i] = byte('a' + i%26)
	}

	opts := ipOptionInformation{TTL: 128, Flags: ipFlagDF}
	replyLen := int(unsafe.Sizeof(icmpEchoReply{})) + payload + 8 + 256
	reply := make([]byte, replyLen)

	replies, _, lastErr := syscall.SyscallN(procIcmpSendEcho.Addr(),
		handle,
		uintptr(dest),
		uintptr(unsafe.Pointer(&data[0])),
		uintptr(uint16(payload)),
		uintptr(unsafe.Pointer(&opts)),
		uintptr(unsafe.Pointer(&reply[0])),
		uintptr(uint32(replyLen)),
		uintptr(uint32(timeout.Milliseconds())),
	)

	if replies == 0 {
		// Sem respostas: o motivo real vem em GetLastError como código IP_STATUS.
		var code uint32 = ipGeneralFailure
		var errno syscall.Errno
		if errors.As(lastErr, &errno) {
			code = uint32(errno)
		}
		return statusFromIP(code)
	}

	r := (*icmpEchoReply)(unsafe.Pointer(&reply[0]))
	return statusFromIP(r.Status)
}

// PingRTT envia um único eco ICMP (sem DF, payload pequeno e fixo — o objetivo
// aqui é medir tempo de ida e volta, não descobrir MTU) e devolve o RTT em ms
// que o próprio Windows calculou (campo RoundTripTime do IcmpSendEcho).
// ok=false quando não houve resposta (timeout, inalcançável) — não é erro, é
// um pacote perdido, que quem chama conta como perda.
func (LiveProber) PingRTT(ctx context.Context, dst netip.Addr, timeout time.Duration) (rttMs int, ok bool, err error) {
	if err := ctx.Err(); err != nil {
		return 0, false, err
	}
	if !dst.Is4() {
		return 0, false, errors.New("medição de latência por enquanto só cobre IPv4")
	}
	for _, p := range []*windows.LazyProc{procIcmpCreateFile, procIcmpCloseHandle, procIcmpSendEcho} {
		if err := p.Find(); err != nil {
			return 0, false, fmt.Errorf("iphlpapi.dll não expôs %s: %w", p.Name, err)
		}
	}

	handle, _, lastErr := syscall.SyscallN(procIcmpCreateFile.Addr())
	if handle == uintptr(windows.InvalidHandle) || handle == 0 {
		return 0, false, fmt.Errorf("abrindo canal ICMP: %w", lastErr)
	}
	defer syscall.SyscallN(procIcmpCloseHandle.Addr(), handle)

	b4 := dst.As4()
	dest := binary.LittleEndian.Uint32(b4[:])

	const payload = 32 // tamanho clássico do `ping` do Windows — suficiente para medir RTT
	data := make([]byte, payload)
	for i := range data {
		data[i] = byte('a' + i%26)
	}

	opts := ipOptionInformation{TTL: 128}
	replyLen := int(unsafe.Sizeof(icmpEchoReply{})) + payload + 8 + 256
	reply := make([]byte, replyLen)

	replies, _, lastErr := syscall.SyscallN(procIcmpSendEcho.Addr(),
		handle,
		uintptr(dest),
		uintptr(unsafe.Pointer(&data[0])),
		uintptr(uint16(payload)),
		uintptr(unsafe.Pointer(&opts)),
		uintptr(unsafe.Pointer(&reply[0])),
		uintptr(uint32(replyLen)),
		uintptr(uint32(timeout.Milliseconds())),
	)

	if replies == 0 {
		return 0, false, nil
	}

	r := (*icmpEchoReply)(unsafe.Pointer(&reply[0]))
	if r.Status != ipSuccess {
		return 0, false, nil
	}
	return int(r.RoundTripTime), true, nil
}
