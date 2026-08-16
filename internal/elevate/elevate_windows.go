//go:build windows

// Package elevate responde à pergunta "este processo é administrador?".
//
// O app NUNCA roda inteiro elevado nem instala serviço sempre-elevado — a
// elevação é sob demanda, em lote (docs/arquitetura-app-desktop.md, seção 3).
// Saber de antemão se somos administrador serve para avisar o usuário ANTES de
// tentar e falhar.
package elevate

import "golang.org/x/sys/windows"

// IsElevated diz se o token do processo atual está elevado.
func IsElevated() bool {
	return windows.GetCurrentProcessToken().IsElevated()
}
