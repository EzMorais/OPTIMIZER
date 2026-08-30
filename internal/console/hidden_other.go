//go:build !windows

package console

import "os/exec"

// HideWindow é no-op fora do Windows.
func HideWindow(_ *exec.Cmd) {}
