package winreg

import (
	"sort"
	"strings"
	"sync"
)

// Fake é um registro em memória para testes. Reproduz o que importa para o
// motor de tweaks: valor ausente, tipo errado e acesso negado (falta de
// elevação em HKLM), sem tocar o Windows real.
type Fake struct {
	mu     sync.Mutex
	values map[string]any // "HKCU\Caminho!Nome" -> uint32 | string
	denied []string       // prefixos que simulam falta de elevação
}

var _ Backend = (*Fake)(nil)

func NewFake() *Fake { return &Fake{values: map[string]any{}} }

// Key monta a chave interna usada tanto pelo Fake quanto pelos snapshots.
func Key(root Root, path, name string) string {
	return string(root) + `\` + path + `!` + name
}

// Seed grava um valor inicial (uint32 para REG_DWORD, string para REG_SZ).
func (f *Fake) Seed(root Root, path, name string, value any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.values[Key(root, path, name)] = value
}

// DenyPrefix faz qualquer escrita/leitura sob esse prefixo retornar
// ErrAccessDenied — usado para testar o caminho "precisa de administrador".
func (f *Fake) DenyPrefix(prefix string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.denied = append(f.denied, prefix)
}

// Dump devolve uma cópia ordenada do conteúdo, para asserções em teste.
func (f *Fake) Dump() map[string]any {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make(map[string]any, len(f.values))
	for k, v := range f.values {
		out[k] = v
	}
	return out
}

// Keys devolve as chaves gravadas, ordenadas.
func (f *Fake) Keys() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]string, 0, len(f.values))
	for k := range f.values {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func (f *Fake) blocked(key string) bool {
	for _, p := range f.denied {
		if strings.HasPrefix(key, p) {
			return true
		}
	}
	return false
}

func (f *Fake) get(root Root, path, name string) (any, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	key := Key(root, path, name)
	if f.blocked(key) {
		return nil, ErrAccessDenied
	}
	v, ok := f.values[key]
	if !ok {
		return nil, ErrNotExist
	}
	return v, nil
}

func (f *Fake) GetDWord(root Root, path, name string) (uint32, error) {
	v, err := f.get(root, path, name)
	if err != nil {
		return 0, err
	}
	n, ok := v.(uint32)
	if !ok {
		return 0, ErrUnexpectedType
	}
	return n, nil
}

func (f *Fake) GetString(root Root, path, name string) (string, error) {
	v, err := f.get(root, path, name)
	if err != nil {
		return "", err
	}
	s, ok := v.(string)
	if !ok {
		return "", ErrUnexpectedType
	}
	return s, nil
}

func (f *Fake) set(root Root, path, name string, value any) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	key := Key(root, path, name)
	if f.blocked(key) {
		return ErrAccessDenied
	}
	f.values[key] = value
	return nil
}

func (f *Fake) SetDWord(root Root, path, name string, value uint32) error {
	return f.set(root, path, name, value)
}

func (f *Fake) SetString(root Root, path, name, value string) error {
	return f.set(root, path, name, value)
}

func (f *Fake) DeleteValue(root Root, path, name string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	key := Key(root, path, name)
	if f.blocked(key) {
		return ErrAccessDenied
	}
	if _, ok := f.values[key]; !ok {
		return ErrNotExist
	}
	delete(f.values, key)
	return nil
}

func (f *Fake) ValueNames(root Root, path string) ([]string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	prefix := string(root) + `\` + path + `!`
	var out []string
	for k := range f.values {
		if strings.HasPrefix(k, prefix) {
			out = append(out, strings.TrimPrefix(k, prefix))
		}
	}
	if out == nil {
		return nil, ErrNotExist
	}
	sort.Strings(out)
	return out, nil
}
