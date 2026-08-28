package systemdiag

import (
	"testing"

	"optimizer/internal/winreg"
)

func TestListarStartupItems(t *testing.T) {
	fake := winreg.NewFake()
	fake.Seed(winreg.HKCU, regRun, "MeuApp", "C:\\Program Files\\App\\app.exe")
	fake.Seed(winreg.HKLM, regRun, "DriverHelper", "C:\\Windows\\System32\\driver.exe")

	itens, err := ListarStartupItems(fake)
	if err != nil {
		t.Fatalf("ListarStartupItems falhou: %v", err)
	}

	if len(itens) < 2 {
		t.Errorf("esperado pelo menos 2 itens, obteve %d", len(itens))
	}

	achouHKCU := false
	for _, it := range itens {
		if it.Nome == "MeuApp" && it.Origem == "HKCU (Usuário)" {
			achouHKCU = true
		}
	}
	if !achouHKCU {
		t.Error("não encontrou MeuApp do HKCU")
	}
}

func TestAlternarStartup(t *testing.T) {
	fake := winreg.NewFake()
	fake.Seed(winreg.HKCU, regRun, "AppTeste", "teste.exe")

	err := AlternarStartup(fake, "hkcu.AppTeste", false)
	if err != nil {
		t.Fatalf("AlternarStartup desativar falhou: %v", err)
	}

	itens, _ := ListarStartupItems(fake)
	for _, it := range itens {
		if it.Nome == "AppTeste" {
			t.Error("AppTeste ainda está presente após desativar")
		}
	}
}
