//go:build windows

// Package restore cria o ponto de restauração do Windows antes de qualquer
// alteração — rede de segurança que, por decisão de produto, é gratuita e
// sempre ligada (docs/decisoes.md).
package restore

import (
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Tipos de evento/restauração do System Restore (srrestoreptapi.h).
const (
	beginSystemChange  = 100
	endSystemChange    = 101
	modifySettings     = 12
	cancelledOperation = 13
)

type restorePointInfo struct {
	EventType      uint32
	RestorePtType  uint32
	SequenceNumber int64
	Description    [256]uint16
}

type stateMgrStatus struct {
	Status         uint32
	SequenceNumber int64
}

var (
	modSrClient            = windows.NewLazySystemDLL("SrClient.dll")
	procSRSetRestorePointW = modSrClient.NewProc("SRSetRestorePointW")
)

func call(info *restorePointInfo) (stateMgrStatus, error) {
	if err := procSRSetRestorePointW.Find(); err != nil {
		return stateMgrStatus{}, fmt.Errorf("SrClient.dll indisponível nesta edição do Windows: %w", err)
	}
	var status stateMgrStatus
	ok, _, lastErr := syscall.SyscallN(procSRSetRestorePointW.Addr(),
		uintptr(unsafe.Pointer(info)), uintptr(unsafe.Pointer(&status)))
	if ok == 0 {
		return status, friendly(status.Status, lastErr)
	}
	return status, nil
}

func friendly(status uint32, lastErr error) error {
	switch syscall.Errno(status) {
	case windows.ERROR_SERVICE_DISABLED:
		return fmt.Errorf("a Proteção do Sistema está desligada neste computador — " +
			"ligue em \"Criar um ponto de restauração\" para que o app possa criar o ponto de segurança")
	case windows.ERROR_ACCESS_DENIED:
		return fmt.Errorf("criar ponto de restauração exige permissão de administrador")
	case 0:
		return fmt.Errorf("não foi possível criar o ponto de restauração: %w", lastErr)
	}
	return fmt.Errorf("não foi possível criar o ponto de restauração (código %d)", status)
}

// Begin abre um ponto de restauração e devolve o número de sequência, que
// precisa ser passado para End depois que as alterações terminarem.
//
// Atenção documentada: o Windows limita a criação a um ponto a cada 24 h por
// padrão (SystemRestorePointCreationFrequency). Quando o limite bate, a chamada
// retorna sucesso sem criar ponto novo — o app não deve prometer ao usuário um
// ponto novo a cada execução.
func Begin(description string) (uint64, error) {
	info := restorePointInfo{EventType: beginSystemChange, RestorePtType: modifySettings}
	copyDescription(&info, description)

	st, err := call(&info)
	if err != nil {
		return 0, err
	}
	return uint64(st.SequenceNumber), nil
}

// End fecha o ponto de restauração aberto por Begin.
func End(sequence uint64) error {
	info := restorePointInfo{
		EventType:      endSystemChange,
		RestorePtType:  modifySettings,
		SequenceNumber: int64(sequence),
	}
	_, err := call(&info)
	return err
}

// Cancel descarta o ponto quando o lote foi abortado antes de mudar qualquer coisa.
func Cancel(sequence uint64) error {
	info := restorePointInfo{
		EventType:      endSystemChange,
		RestorePtType:  cancelledOperation,
		SequenceNumber: int64(sequence),
	}
	_, err := call(&info)
	return err
}

func copyDescription(info *restorePointInfo, s string) {
	if s == "" {
		s = "Optimizer"
	}
	u, err := windows.UTF16FromString(s)
	if err != nil {
		u = windows.StringToUTF16("Optimizer")
	}
	if len(u) > len(info.Description) {
		u = u[:len(info.Description)]
		u[len(u)-1] = 0
	}
	copy(info.Description[:], u)
}
