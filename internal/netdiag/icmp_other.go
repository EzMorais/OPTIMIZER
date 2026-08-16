//go:build !windows

package netdiag

import (
	"context"
	"errors"
	"net/netip"
	"time"
)

// LiveProber fora do Windows existe só para o pacote compilar; a medição real
// depende de IcmpSendEcho (iphlpapi.dll).
type LiveProber struct{}

var _ Prober = LiveProber{}

func (LiveProber) Ping(context.Context, netip.Addr, int, time.Duration) (ProbeStatus, error) {
	return ProbeTimeout, errors.New("netdiag: medição de MTU disponível apenas no Windows")
}
