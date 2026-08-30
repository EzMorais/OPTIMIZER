package systemdiag_test

import (
	"context"
	"testing"

	"optimizer/internal/systemdiag"
	"optimizer/internal/winreg"
)

type fakeMsiRunner struct {
	output string
	err    error
}

func (f *fakeMsiRunner) ExecPowerShell(ctx context.Context, script string) ([]byte, error) {
	if f.err != nil {
		return nil, f.err
	}
	return []byte(f.output), nil
}

func TestListarDispositivosMSI(t *testing.T) {
	jsonSample := `[{"InstanceId":"PCI\\VEN_10DE&DEV_1F02\\0001","FriendlyName":"NVIDIA GeForce RTX 2060","Class":"Display","MSISupported":1,"RegPath":"SYSTEM\\CurrentControlSet\\Enum\\PCI\\VEN_10DE&DEV_1F02\\0001\\Device Parameters\\Interrupt Management\\MessageSignaledInterruptProperties"}]`
	systemdiag.SetMsiRunner(&fakeMsiRunner{output: jsonSample})

	devs, err := systemdiag.ListarDispositivosMSI(context.Background())
	if err != nil {
		t.Fatalf("ListarDispositivosMSI falhou: %v", err)
	}
	if len(devs) != 1 {
		t.Fatalf("Esperava 1 dispositivo MSI, encontrou %d", len(devs))
	}
	if !devs[0].MSISupported {
		t.Errorf("Esperava MSISupported = true, obteve false")
	}
	if devs[0].Nome != "NVIDIA GeForce RTX 2060" {
		t.Errorf("Nome incorreto: %s", devs[0].Nome)
	}
}

func TestAlternarModoMSI(t *testing.T) {
	fake := winreg.NewFake()
	msiKey := `SYSTEM\CurrentControlSet\Enum\PCI\VEN_10DE&DEV_1F02\0001\Device Parameters\Interrupt Management\MessageSignaledInterruptProperties`

	if err := systemdiag.AlternarModoMSI(fake, msiKey, true); err != nil {
		t.Fatalf("AlternarModoMSI falhou: %v", err)
	}
	val, err := fake.GetDWord(winreg.HKLM, msiKey, "MSISupported")
	if err != nil || val != 1 {
		t.Errorf("Valor MSISupported esperado 1, obteve %d (err: %v)", val, err)
	}

	if err := systemdiag.AlternarModoMSI(fake, msiKey, false); err != nil {
		t.Fatalf("AlternarModoMSI (desativar) falhou: %v", err)
	}
	val, err = fake.GetDWord(winreg.HKLM, msiKey, "MSISupported")
	if err != nil || val != 0 {
		t.Errorf("Valor MSISupported esperado 0, obteve %d (err: %v)", val, err)
	}
}
