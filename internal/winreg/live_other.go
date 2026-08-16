//go:build !windows

package winreg

// Live fora do Windows existe só para o pacote compilar (e os testes com Fake
// rodarem em qualquer plataforma). Toda chamada falha explicitamente.
type Live struct{}

func NewLive() Live { return Live{} }

var _ Backend = Live{}

func (Live) GetDWord(Root, string, string) (uint32, error)  { return 0, ErrUnsupported }
func (Live) GetString(Root, string, string) (string, error) { return "", ErrUnsupported }
func (Live) SetDWord(Root, string, string, uint32) error    { return ErrUnsupported }
func (Live) SetString(Root, string, string, string) error   { return ErrUnsupported }
func (Live) DeleteValue(Root, string, string) error         { return ErrUnsupported }
func (Live) ValueNames(Root, string) ([]string, error)      { return nil, ErrUnsupported }
