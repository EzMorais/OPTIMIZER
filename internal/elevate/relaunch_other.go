//go:build !windows

package elevate

import "errors"

// Relaunch fora do Windows não existe: não há UAC.
func Relaunch() error { return errors.New("elevate: elevação só existe no Windows") }
