//go:build windows

package winreg

import (
	"errors"
	"fmt"
	"syscall"

	"golang.org/x/sys/windows/registry"
)

// Live fala com o registro real via golang.org/x/sys/windows/registry
// (API oficial, sem processo externo — ver arquitetura, seção 2).
type Live struct{}

// NewLive devolve o backend que toca o registro de verdade.
func NewLive() Live { return Live{} }

var _ Backend = Live{}

func hive(r Root) (registry.Key, error) {
	switch r {
	case HKCU:
		return registry.CURRENT_USER, nil
	case HKLM:
		return registry.LOCAL_MACHINE, nil
	default:
		return 0, fmt.Errorf("winreg: raiz desconhecida %q", r)
	}
}

// translate normaliza erros do Windows para os erros sentinela do pacote,
// para que o motor de tweaks não precise conhecer códigos Win32.
func translate(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, registry.ErrNotExist), errors.Is(err, syscall.ERROR_PATH_NOT_FOUND):
		return ErrNotExist
	case errors.Is(err, syscall.ERROR_ACCESS_DENIED):
		return ErrAccessDenied
	case errors.Is(err, registry.ErrUnexpectedType):
		return ErrUnexpectedType
	default:
		return err
	}
}

func (Live) open(root Root, path string, access uint32) (registry.Key, error) {
	h, err := hive(root)
	if err != nil {
		return 0, err
	}
	k, err := registry.OpenKey(h, path, access)
	if err != nil {
		return 0, translate(err)
	}
	return k, nil
}

func (l Live) create(root Root, path string) (registry.Key, error) {
	h, err := hive(root)
	if err != nil {
		return 0, err
	}
	k, _, err := registry.CreateKey(h, path, registry.SET_VALUE)
	if err != nil {
		return 0, translate(err)
	}
	return k, nil
}

func (l Live) GetDWord(root Root, path, name string) (uint32, error) {
	k, err := l.open(root, path, registry.QUERY_VALUE)
	if err != nil {
		return 0, err
	}
	defer k.Close()

	v, _, err := k.GetIntegerValue(name)
	if err != nil {
		return 0, translate(err)
	}
	return uint32(v), nil
}

func (l Live) GetString(root Root, path, name string) (string, error) {
	k, err := l.open(root, path, registry.QUERY_VALUE)
	if err != nil {
		return "", err
	}
	defer k.Close()

	v, _, err := k.GetStringValue(name)
	if err != nil {
		return "", translate(err)
	}
	return v, nil
}

func (l Live) SetDWord(root Root, path, name string, value uint32) error {
	k, err := l.create(root, path)
	if err != nil {
		return err
	}
	defer k.Close()
	return translate(k.SetDWordValue(name, value))
}

func (l Live) SetString(root Root, path, name, value string) error {
	k, err := l.create(root, path)
	if err != nil {
		return err
	}
	defer k.Close()
	return translate(k.SetStringValue(name, value))
}

func (l Live) DeleteValue(root Root, path, name string) error {
	k, err := l.open(root, path, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	return translate(k.DeleteValue(name))
}

func (l Live) ValueNames(root Root, path string) ([]string, error) {
	k, err := l.open(root, path, registry.QUERY_VALUE|registry.ENUMERATE_SUB_KEYS)
	if err != nil {
		return nil, err
	}
	defer k.Close()

	names, err := k.ReadValueNames(0)
	if err != nil {
		return nil, translate(err)
	}
	return names, nil
}
