package restore

import (
	"errors"
	"strings"
	"syscall"
	"testing"

	"golang.org/x/sys/windows"
)

func TestCopyDescription(t *testing.T) {
	var info restorePointInfo

	// Descrição normal
	copyDescription(&info, "Meu Ponto de Restauração")
	str := windows.UTF16ToString(info.Description[:])
	if str != "Meu Ponto de Restauração" {
		t.Errorf("esperado 'Meu Ponto de Restauração', obteve %q", str)
	}

	// Descrição vazia cai para padrão "Optimizer"
	copyDescription(&info, "")
	str = windows.UTF16ToString(info.Description[:])
	if str != "Optimizer" {
		t.Errorf("esperado 'Optimizer' para string vazia, obteve %q", str)
	}

	// Descrição muito longa é truncada com terminação nula segura
	longa := strings.Repeat("A", 300)
	copyDescription(&info, longa)
	if info.Description[len(info.Description)-1] != 0 {
		t.Error("esperado terminador nulo no final do buffer")
	}
}

func TestFriendlyErrors(t *testing.T) {
	// 1. Serviço desabilitado
	errDisabled := friendly(uint32(windows.ERROR_SERVICE_DISABLED), nil)
	if !strings.Contains(errDisabled.Error(), "Proteção do Sistema está desligada") {
		t.Errorf("mensagem inesperada para ERROR_SERVICE_DISABLED: %v", errDisabled)
	}

	// 2. Acesso negado
	errDenied := friendly(uint32(windows.ERROR_ACCESS_DENIED), nil)
	if !strings.Contains(errDenied.Error(), "administrador") {
		t.Errorf("mensagem inesperada para ERROR_ACCESS_DENIED: %v", errDenied)
	}

	// 3. Erro genérico
	lastErr := errors.New("falha de driver")
	errGeneric := friendly(0, lastErr)
	if !strings.Contains(errGeneric.Error(), "falha de driver") {
		t.Errorf("mensagem inesperada para erro 0 com lastErr: %v", errGeneric)
	}

	// 4. Código desconhecido
	errCode := friendly(9999, nil)
	if !strings.Contains(errCode.Error(), "código 9999") {
		t.Errorf("mensagem inesperada para código: %v", errCode)
	}

	_ = syscall.Errno(0)
}
