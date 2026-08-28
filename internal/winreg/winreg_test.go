package winreg_test

import (
	"errors"
	"testing"

	"optimizer/internal/winreg"
)

func TestFakeDWordOperations(t *testing.T) {
	f := winreg.NewFake()
	path := `Software\Test`
	name := "DWordVal"

	// 1. Inexistente
	_, err := f.GetDWord(winreg.HKCU, path, name)
	if !errors.Is(err, winreg.ErrNotExist) {
		t.Fatalf("esperado ErrNotExist, obteve: %v", err)
	}

	// 2. Set
	if err := f.SetDWord(winreg.HKCU, path, name, 42); err != nil {
		t.Fatalf("SetDWord falhou: %v", err)
	}

	// 3. Get
	val, err := f.GetDWord(winreg.HKCU, path, name)
	if err != nil {
		t.Fatalf("GetDWord falhou: %v", err)
	}
	if val != 42 {
		t.Errorf("esperado 42, obteve %d", val)
	}

	// 4. Tipo incompatível
	_, err = f.GetString(winreg.HKCU, path, name)
	if !errors.Is(err, winreg.ErrUnexpectedType) {
		t.Errorf("esperado ErrUnexpectedType, obteve: %v", err)
	}

	// 5. Delete
	if err := f.DeleteValue(winreg.HKCU, path, name); err != nil {
		t.Fatalf("DeleteValue falhou: %v", err)
	}

	// 6. Delete após remoção
	if err := f.DeleteValue(winreg.HKCU, path, name); !errors.Is(err, winreg.ErrNotExist) {
		t.Errorf("esperado ErrNotExist ao deletar novamente, obteve: %v", err)
	}
}

func TestFakeStringOperations(t *testing.T) {
	f := winreg.NewFake()
	path := `Software\Test`
	name := "StrVal"

	if err := f.SetString(winreg.HKLM, path, name, "hello"); err != nil {
		t.Fatalf("SetString falhou: %v", err)
	}

	val, err := f.GetString(winreg.HKLM, path, name)
	if err != nil {
		t.Fatalf("GetString falhou: %v", err)
	}
	if val != "hello" {
		t.Errorf("esperado 'hello', obteve %q", val)
	}
}

func TestFakeDenyPrefix(t *testing.T) {
	f := winreg.NewFake()
	f.DenyPrefix("HKLM")

	err := f.SetDWord(winreg.HKLM, `Software\Test`, "Val", 1)
	if !errors.Is(err, winreg.ErrAccessDenied) {
		t.Errorf("esperado ErrAccessDenied para prefixo bloqueado, obteve: %v", err)
	}
}

func TestFakeValueNames(t *testing.T) {
	f := winreg.NewFake()
	path := `Software\TestList`
	f.Seed(winreg.HKCU, path, "Beta", uint32(1))
	f.Seed(winreg.HKCU, path, "Alpha", "abc")

	names, err := f.ValueNames(winreg.HKCU, path)
	if err != nil {
		t.Fatalf("ValueNames falhou: %v", err)
	}
	if len(names) != 2 || names[0] != "Alpha" || names[1] != "Beta" {
		t.Errorf("ValueNames inesperado: %v", names)
	}
}
