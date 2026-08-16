// Package winreg isola o acesso ao registro do Windows atrás de uma interface.
//
// Motivo: Check/Apply/Revert de todo tweak precisam ser testáveis sem tocar o
// registro real da máquina de desenvolvimento — ver docs/arquitetura-app-desktop.md,
// seção 7, item 1 ("abstrair a fronteira"). A implementação real (Live) só existe
// no Windows; os testes usam Fake, em memória, e rodam em qualquer plataforma.
package winreg

import "errors"

// Root é a raiz da chave. Só expomos as duas que o catálogo usa: nada de
// HKEY_CLASSES_ROOT/HKEY_USERS enquanto não houver caso de uso real.
type Root string

const (
	HKCU Root = "HKCU"
	HKLM Root = "HKLM"
)

var (
	// ErrNotExist cobre tanto chave inexistente quanto valor inexistente —
	// para o motor de tweaks os dois casos significam "Windows no padrão".
	ErrNotExist = errors.New("winreg: chave ou valor não existe")
	// ErrAccessDenied normalmente significa que falta elevação (típico de HKLM).
	ErrAccessDenied = errors.New("winreg: acesso negado — requer privilégio de administrador")
	// ErrUnsupported é retornado pela implementação real fora do Windows.
	ErrUnsupported = errors.New("winreg: registro do Windows indisponível nesta plataforma")
	// ErrUnexpectedType indica que o valor existe mas com outro tipo (ex.: REG_SZ onde
	// esperávamos REG_DWORD). Nunca sobrescrevemos às cegas nesse caso.
	ErrUnexpectedType = errors.New("winreg: valor existe com tipo diferente do esperado")
)

// Backend é a fronteira mockável com o registro.
type Backend interface {
	GetDWord(root Root, path, name string) (uint32, error)
	GetString(root Root, path, name string) (string, error)
	SetDWord(root Root, path, name string, value uint32) error
	SetString(root Root, path, name, value string) error
	// DeleteValue remove o valor. Retorna ErrNotExist se ele já não existia.
	DeleteValue(root Root, path, name string) error
	// ValueNames lista os nomes de valores de uma chave (usado por inicialização).
	ValueNames(root Root, path string) ([]string, error)
}
