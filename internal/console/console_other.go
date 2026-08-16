//go:build !windows

package console

// EnableUTF8 não faz nada fora do Windows: os terminais já são UTF-8.
func EnableUTF8() {}
