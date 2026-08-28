package systemdiag

import (
	"context"
	"fmt"
	"testing"
)

type mockPnpRunner struct {
	psOutput  string
	psErr     error
	removed   []string
	removeErr error
}

func (m *mockPnpRunner) ExecPowerShell(ctx context.Context, script string) ([]byte, error) {
	if m.psErr != nil {
		return nil, m.psErr
	}
	return []byte(m.psOutput), nil
}

func (m *mockPnpRunner) RemoveDevice(ctx context.Context, instanceID string) error {
	if m.removeErr != nil {
		return m.removeErr
	}
	m.removed = append(m.removed, instanceID)
	return nil
}

func TestListarDispositivosFantasmas(t *testing.T) {
	mock := &mockPnpRunner{
		psOutput: `[
			{"InstanceId":"USB\\VID_046D&PID_C52B\\1234","FriendlyName":"Logitech USB Receiver Antigo","Class":"Mouse","Status":"Unknown","Present":false},
			{"InstanceId":"PCI\\VEN_10DE&DEV_1B81\\5678","FriendlyName":"GPU Antiga Desconectada","Class":"Display","Status":"Unknown","Present":false}
		]`,
	}
	SetPnpRunner(mock)
	defer SetPnpRunner(&defaultPnpRunner{})

	devs, err := ListarDispositivosFantasmas(context.Background())
	if err != nil {
		t.Fatalf("ListarDispositivosFantasmas erro: %v", err)
	}
	if len(devs) != 2 {
		t.Fatalf("esperado 2 dispositivos, obteve %d", len(devs))
	}
	if devs[0].FriendlyName != "Logitech USB Receiver Antigo" {
		t.Errorf("FriendlyName = %s, esperado Logitech USB Receiver Antigo", devs[0].FriendlyName)
	}
}

func TestListarDispositivosFantasmasUnico(t *testing.T) {
	mock := &mockPnpRunner{
		psOutput: `{"InstanceId":"USB\\VID_0000&PID_0000\\1","FriendlyName":"","Class":"USB","Status":"Unknown","Present":false}`,
	}
	SetPnpRunner(mock)
	defer SetPnpRunner(&defaultPnpRunner{})

	devs, err := ListarDispositivosFantasmas(context.Background())
	if err != nil {
		t.Fatalf("ListarDispositivosFantasmas erro: %v", err)
	}
	if len(devs) != 1 {
		t.Fatalf("esperado 1 dispositivo, obteve %d", len(devs))
	}
	if devs[0].FriendlyName != "USB\\VID_0000&PID_0000\\1" {
		t.Errorf("esperado fallback para InstanceId, obteve %s", devs[0].FriendlyName)
	}
}

func TestLimparDispositivosFantasmas(t *testing.T) {
	mock := &mockPnpRunner{}
	SetPnpRunner(mock)
	defer SetPnpRunner(&defaultPnpRunner{})

	ids := []string{"USB\\VID_123\\1", "PCI\\VEN_456\\2"}
	removidos, errs, err := LimparDispositivosFantasmas(context.Background(), ids)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if removidos != 2 || len(errs) != 0 {
		t.Fatalf("esperado 2 removidos e 0 erros, obteve %d removidos e %d erros", removidos, len(errs))
	}
	if len(mock.removed) != 2 {
		t.Errorf("mock.removed tamanho = %d, esperado 2", len(mock.removed))
	}
}

func TestLimparDispositivosFantasmasComFalha(t *testing.T) {
	mock := &mockPnpRunner{
		removeErr: fmt.Errorf("acesso negado"),
	}
	SetPnpRunner(mock)
	defer SetPnpRunner(&defaultPnpRunner{})

	ids := []string{"USB\\VID_123\\1"}
	removidos, errs, err := LimparDispositivosFantasmas(context.Background(), ids)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if removidos != 0 || len(errs) != 1 {
		t.Fatalf("esperado 0 removidos e 1 erro, obteve %d e %d", removidos, len(errs))
	}
}
