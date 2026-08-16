//go:build windows

// Package console ajusta o terminal para exibir texto em português corretamente.
package console

import "golang.org/x/sys/windows"

var (
	modkernel32            = windows.NewLazySystemDLL("kernel32.dll")
	procSetConsoleOutputCP = modkernel32.NewProc("SetConsoleOutputCP")
	procSetConsoleCP       = modkernel32.NewProc("SetConsoleCP")
)

const codePageUTF8 = 65001

// EnableUTF8 evita acento quebrado no Prompt de Comando/PowerShell, que ainda
// abrem com a página de código antiga do Windows.
func EnableUTF8() {
	if err := procSetConsoleOutputCP.Find(); err == nil {
		procSetConsoleOutputCP.Call(codePageUTF8)
	}
	if err := procSetConsoleCP.Find(); err == nil {
		procSetConsoleCP.Call(codePageUTF8)
	}
}
