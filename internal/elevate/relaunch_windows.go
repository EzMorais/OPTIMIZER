//go:build windows

package elevate

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/windows"
)

// Relaunch reabre o próprio executável pedindo elevação ao Windows (o diálogo
// do UAC). O processo atual deve encerrar em seguida — é o padrão "elevação sob
// demanda" da arquitetura: nunca existe um processo elevado permanente.
func Relaunch() error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("descobrindo o próprio executável: %w", err)
	}
	verb, err := windows.UTF16PtrFromString("runas")
	if err != nil {
		return err
	}
	file, err := windows.UTF16PtrFromString(exe)
	if err != nil {
		return err
	}
	dir, err := windows.UTF16PtrFromString(filepath.Dir(exe))
	if err != nil {
		return err
	}

	const swNormal = 1
	if err := windows.ShellExecute(0, verb, file, nil, dir, swNormal); err != nil {
		return fmt.Errorf("o Windows recusou a elevação: %w", err)
	}
	return nil
}
